// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package device

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// This file is the read loop: the one goroutine that reads from a device, and
// the waits everything else does instead of reading.
//
// A device sends notifications unasked. With nothing draining the endpoint
// its outgoing queue fills, at which point it stops draining the incoming one
// too and the next write times out. So a read is posted from the claim until
// Close, including between operations, during the flash pause and while a
// switch is polled.

// start begins reading. Everything after it waits rather than reads.
func (s *session) start() {
	ctx, cancel := context.WithCancel(context.Background())

	s.stop = cancel
	s.loopDone = make(chan struct{})
	s.ackDone = make(chan struct{})

	go s.loop(ctx)
	go s.acknowledge(ctx)
}

// loop reads until ctx ends or the bus fails.
func (s *session) loop(
	ctx context.Context,
) {
	defer close(s.loopDone)

	// A panic on a goroutine nobody joins kills the process without running
	// the caller's deferred Close, and that leaves the pedal needing a power
	// cycle. So it ends the session the way a bus error does instead.
	defer func() {
		if v := recover(); v != nil {
			s.end(&busError{
				err: fmt.Errorf("the read loop panicked: %v\n%s", v, debug.Stack()),
			})
		}
	}()

	buf := make([]byte, readBuffer)

	for ctx.Err() == nil {
		if err := s.readOnce(ctx, buf); err != nil {
			s.end(err)

			return
		}
	}
}

// readOnce posts one read for a window and routes whatever it brings.
//
// A read that timed out is quiet, not a failure. Any other failure is the
// bus, and ends the session: read as silence, it would wait out every budget
// and then blame the device.
func (s *session) readOnce(
	ctx context.Context,
	buf []byte,
) error {
	window, cancel := context.WithTimeout(ctx, s.budgets.window)
	defer cancel()

	n, err := s.in.ReadContext(window, buf)

	switch {
	case err == nil:
		s.route(buf[:n])
	case ctx.Err() != nil:
		// Close stopped the loop, which is not a failure.
	case errors.Is(err, context.DeadlineExceeded):
		// The ordinary case: the device had nothing to say in time. One that
		// says so before the window is up would spin the loop, so the rest of
		// the window is waited out.
		<-window.Done()
		s.tick(false)
	default:
		return &busError{err: fmt.Errorf("reading from the device: %w", err)}
	}

	return nil
}

// tick records a finished read and wakes whoever is waiting on one.
func (s *session) tick(
	transfer bool,
) {
	s.advance(transfer)

	// One slot: a pending poke already says a read finished.
	select {
	case s.poke <- struct{}{}:
	default:
	}
}

// advance counts a finished read.
func (s *session) advance(
	transfer bool,
) {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	s.windows++

	if transfer {
		s.transfers++
		s.quietRun = 0
	} else {
		s.quietRun++
	}

	close(s.changed)
	s.changed = make(chan struct{})
}

// end records what ended the loop, once, and tells every waiter.
func (s *session) end(
	err error,
) {
	s.endOnce.Do(func() {
		s.endErr = err
		close(s.dead)
	})
}

// ended is what ended the loop, or nil while it runs.
func (s *session) ended() error {
	select {
	case <-s.dead:
		return s.endErr
	default:
		return nil
	}
}

// progress is where the loop has got to.
type progress struct {
	// windows counts finished reads, transfers those that brought something,
	// and quiet how many in a row brought nothing.
	windows   uint64
	transfers uint64
	quiet     int
	// next closes when the next read finishes.
	next <-chan struct{}
}

// progress reports where the loop has got to.
func (s *session) progress() progress {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	return progress{
		windows:   s.windows,
		transfers: s.transfers,
		quiet:     s.quietRun,
		next:      s.changed,
	}
}

// acknowledge settles what arrives on a channel nobody is using.
//
// A goroutine of its own rather than work the loop does, so a slow write
// never holds a read back.
func (s *session) acknowledge(
	ctx context.Context,
) {
	defer close(s.ackDone)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.dead:
			return
		case <-s.poke:
		}

		for _, c := range s.opened() {
			s.idleAck(c)
		}
	}
}

// idleAck acknowledges a channel whose unasked-for bytes have gone quiet.
//
// At most once a quiet period, since the acknowledgement itself settles what
// is owed. The send lock is taken before the channel is looked at, so an
// exchange cannot start between deciding and sending.
func (s *session) idleAck(
	c *channel,
) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if !s.idleDue(c) {
		return
	}

	// Nobody is waiting on this, so the trace is the one place it shows. A
	// bus that has really gone fails the next operation's send.
	if err := s.sendLocked(c, wire.MsgAck, nil); err != nil {
		s.tracef("ERR %-8s idle ack: %v\n", c.name, err)
	}
}

// idleDue reports a channel owed an idle acknowledgement, and drops the
// complete envelopes nobody asked for.
//
// A partial envelope stays until the rest of it arrives, because the framing
// has no marker to resynchronise on.
func (s *session) idleDue(
	c *channel,
) bool {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	if s.closing || c.busy || !c.owed() || time.Since(c.lastRx) < s.budgets.idle {
		return false
	}

	dropMessages(c)

	return true
}

// dropMessages throws away every complete envelope a channel holds. The
// caller holds the receive lock.
func dropMessages(
	c *channel,
) {
	for {
		if _, ok := message(c); !ok {
			return
		}
	}
}

// pause waits for the device to say anything after mark, or for wait to
// pass.
//
// Silence is not a failure: the device answers some openings and not others.
// A caller who stops waiting is told so, and a loop that ended is the bus.
func (s *session) pause(
	ctx context.Context,
	mark uint64,
	wait time.Duration,
) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	for {
		at := s.progress()
		if at.transfers > mark {
			return nil
		}

		select {
		case <-at.next:
		case <-timer.C:
			return nil
		case <-s.dead:
			return s.endErr
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// pace waits between two chunks of a message for the device to say
// something, for at most the pace budget.
//
// Nothing ends it early but the device. A loop that ended leaves the full
// budget to wait out, so a message never goes out in a burst.
func (s *session) pace(
	mark uint64,
) {
	timer := time.NewTimer(s.budgets.pace)
	defer timer.Stop()

	for {
		at := s.progress()
		if at.transfers > mark {
			return
		}

		select {
		case <-at.next:
		case <-timer.C:
			return
		}
	}
}

// drain waits until the device genuinely has nothing left, and throws away
// what it said.
//
// Three reads in a row with no transfer is a device with nothing left.
// Bounded on purpose. A stale backlog clears in about a hundred frames; an
// unbounded drain keeps the endpoint under load and has coincided with
// devices locking up.
func (s *session) drain(
	ctx context.Context,
) {
	timer := time.NewTimer(s.budgets.drain)
	defer timer.Stop()

	start := s.progress().windows

	for ctx.Err() == nil && s.ended() == nil {
		at := s.progress()
		if at.quiet >= drainQuietRuns && at.windows-start >= drainQuietRuns {
			break
		}

		select {
		case <-at.next:
			continue
		case <-timer.C:
		case <-s.dead:
		case <-ctx.Done():
		}

		break
	}

	// Whatever arrived is consumed, not replayed into a later reply.
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	for _, c := range s.chans {
		c.buf = nil
	}
}
