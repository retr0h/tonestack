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

package tools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// idleClose is how long the pedal stays held after the last device call.
//
// Long enough that an agent's list, show and select share one handshake, and
// short enough that the front panel works again soon after a burst: a pedal
// with an editor attached stops refreshing its footswitches.
const idleClose = 10 * time.Second

// pedal holds one Session across device tools.
//
// It opens a Session on the first device call, hands it to every call after
// that, and closes it when the server stops or once idle passes with no call.
// A call that fails with a bus error closes it too and reports the error, and
// the next call opens a fresh one. That is the agent choosing to call again,
// not a reconnect: nothing here retries.
type pedal struct {
	client Client
	idle   time.Duration
	// lock is taken for every device call and for every close, so two calls
	// never claim the editor interface at once and an idle close never lands
	// inside a call.
	lock chan struct{}
	// session, timer and armed change only under lock.
	session Session
	timer   *time.Timer
	// armed counts every arming and disarming, so an idle close that fired
	// late leaves alone a Session a later call has used.
	armed uint64
	// closed is set once the server has let the pedal go. No call opens it
	// again after that.
	closed bool
}

// errStopped is a device call that arrives after the server let the pedal go.
var errStopped = errors.New("the server has stopped, so the pedal is not reachable")

// newPedal holds nothing until the first device call.
func newPedal(
	c Client,
	idle time.Duration,
) *pedal {
	return &pedal{client: c, idle: idle, lock: make(chan struct{}, 1)}
}

// take claims the pedal for one call, or gives up when the call does.
func (p *pedal) take(
	ctx context.Context,
) error {
	select {
	case p.lock <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("waiting for the device: %w", ctx.Err())
	}

	if p.closed {
		p.give()

		return errStopped
	}

	return nil
}

// give lets the pedal go for the next call.
func (p *pedal) give() { <-p.lock }

// locked runs call while holding the pedal, without a Session.
//
// For listing the bus, which claims nothing but is still kept from running
// beside a call that does.
func locked[T any](
	ctx context.Context,
	p *pedal,
	call func() (T, error),
) (T, error) {
	if err := p.take(ctx); err != nil {
		var zero T

		return zero, err
	}

	defer p.give()

	return call()
}

// onPedal runs call on the held Session, opening one first when none is held.
func onPedal[T any](
	ctx context.Context,
	p *pedal,
	call func(Session) (T, error),
) (T, error) {
	var zero T

	if err := p.take(ctx); err != nil {
		return zero, err
	}

	defer p.give()

	if p.session == nil {
		s, err := p.client.Open(ctx)
		if err != nil {
			return zero, err
		}

		p.session = s
	}

	p.disarm()

	// Armed again however the call ends, so a held Session is always on its
	// way to being let go.
	defer p.arm()

	// go-sdk does not recover a handler that panics. The Session may be
	// partway through an exchange, so it is let go before the panic carries
	// on, rather than left holding the pedal until the process dies. What
	// Close says is lost to the panic.
	defer func() {
		if v := recover(); v != nil {
			_ = p.release()

			panic(v)
		}
	}()

	out, err := call(p.session)

	if errors.Is(err, sdk.ErrBus) {
		// The Session is finished. What Close says is the bus error this call
		// already reports.
		_ = p.release()
	}

	return out, err
}

// arm starts the idle close for the held Session. The caller holds the lock.
func (p *pedal) arm() {
	if p.session == nil {
		return
	}

	p.armed++
	n := p.armed

	p.timer = time.AfterFunc(p.idle, func() { p.expire(n) })
}

// disarm stops the idle close. The caller holds the lock.
func (p *pedal) disarm() {
	p.armed++

	if p.timer != nil {
		p.timer.Stop()
	}
}

// expire lets the pedal go once idle has passed, unless a call has come since.
func (p *pedal) expire(
	n uint64,
) {
	p.lock <- struct{}{}
	defer p.give()

	// Nobody is waiting on this. A Session whose loop ended already told the
	// call it failed, and the next call opens a fresh one either way.
	if n == p.armed {
		_ = p.release()
	}
}

// release closes the held Session, if there is one. The caller holds the
// lock.
func (p *pedal) release() error {
	if p.session == nil {
		return nil
	}

	err := p.session.Close()
	p.session = nil

	return err
}

// Close lets the pedal go, and returns what closing its Session reported.
//
// It waits for a call in flight to finish, which is finite because every
// exchange with the device is bounded.
func (p *pedal) Close() error {
	p.lock <- struct{}{}
	defer p.give()

	p.closed = true
	p.disarm()

	return p.release()
}
