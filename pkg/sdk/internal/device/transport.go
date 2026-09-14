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
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// send writes one frame on a channel and advances its counter.
//
// Every frame the host sends advances the sequence, acknowledgements and
// keep-alives included.
func (s *session) send(c *channel, msgType uint16, payload []byte) error {
	flags := wire.FlagNormal
	ack := c.ack()

	if msgType == wire.MsgHello && payload != nil {
		flags = wire.FlagHandshake
		ack = helloAck
	}

	raw := wire.EncodeFrame(wire.Frame{
		Flags: flags, DeviceNode: c.device, HostNode: c.host,
		Seq: c.seq, Type: msgType, Ack: ack, Payload: payload,
	})

	if s.trace != nil {
		fmt.Fprintf(s.trace, "OUT %-8s seq=%d type=%#04x ack=%#x: %x\n",
			c.name, c.seq, msgType, ack, raw[:min(len(raw), 40)])
	}

	if _, err := s.out.Write(raw); err != nil {
		return fmt.Errorf("writing to %s: %w", c.name, err)
	}

	c.seq++

	return nil
}

// receive reads one transfer and routes every frame in it.
//
// Reports whether any stream bytes arrived, which is what decides if an
// acknowledgement is owed: the device sends empty transfers when it has
// nothing to say, and acknowledging one burns a sequence number and
// desynchronises the channel.
//
// A read that timed out is quiet, not a failure. A caller who stopped waiting
// is told so, and any other failure is the bus, returned rather than read as
// silence: silence waits out a whole budget and then blames the device.
func (s *session) receive(
	ctx context.Context,
	wait time.Duration,
) (bool, error) {
	rctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	buf := make([]byte, readBuffer)

	n, err := s.in.ReadContext(rctx, buf)

	switch {
	case err == nil:
	case ctx.Err() != nil:
		return false, ctx.Err()
	case errors.Is(err, context.DeadlineExceeded):
		// The ordinary case: the device had nothing to say in time. A device
		// is asked far more often than it answers.
		return false, nil
	default:
		return false, fmt.Errorf("reading from the device: %w", err)
	}

	if s.trace != nil {
		fmt.Fprintf(s.trace, "IN  %d bytes: %x\n", n, buf[:min(n, 48)])
	}

	var got bool

	rest := buf[:n]

	for len(rest) > 0 {
		f, remainder, err := wire.DecodeFrame(rest)
		if err != nil {
			// A partial frame at the end of a transfer is not corruption,
			// it is the transfer ending.
			break
		}

		rest = remainder

		if !f.CarriesData() || len(f.Payload) == 0 {
			continue
		}

		c := s.channelFor(f)
		if c == nil {
			continue
		}

		c.rxBytes += uint32(len(f.Payload))
		c.buf = append(c.buf, f.Payload...)
		got = true
	}

	return got, nil
}

// channelFor finds which conversation a frame belongs to.
//
// The device swaps the node fields, so its frames carry the host node where
// a host frame carries the device node.
func (s *session) channelFor(f wire.Frame) *channel {
	for _, c := range s.chans {
		if f.HostNode == c.device || f.DeviceNode == c.device {
			return c
		}
	}

	return nil
}

// drain reads until the device genuinely has nothing left.
//
// Bounded on purpose. A stale backlog clears in about a hundred frames; an
// unbounded drain keeps the endpoint under load and has coincided with
// devices locking up.
func (s *session) drain(
	ctx context.Context,
) {
	deadline := time.Now().Add(drainBudget)

	quiet := 0

	for quiet < drainQuietRuns && time.Now().Before(deadline) {
		// Somebody who stopped waiting is not owed a drained endpoint.
		if ctx.Err() != nil {
			break
		}

		got, err := s.receive(ctx, drainReadWait)
		if err != nil {
			// A bus that has gone is not a quiet device. Reading on would
			// spin against it for the whole budget.
			break
		}

		if got {
			quiet = 0

			continue
		}

		quiet++
	}

	// Whatever arrived is consumed, not replayed into a later reply.
	for _, c := range s.chans {
		c.buf = nil
	}
}

// message takes one complete envelope out of a channel's buffer.
//
// A reply is a byte stream split across frames at 256 bytes each, so it is
// complete only once the declared length has arrived.
func message(c *channel) ([]byte, bool) {
	env, _, err := wire.DecodeEnvelope(c.buf)
	if err != nil {
		// A frame that has not all arrived is waited for. A length no frame
		// carries is not: the buffer is out of step with the stream and no
		// further byte will bring it back, so waiting means every later call
		// on this channel times out with a full buffer nobody can read.
		//
		// Dropped rather than scanned forward. The framing has no marker to
		// resynchronise on, so what is held is unreadable by definition.
		if errors.Is(err, wire.ErrBodyTooLarge) {
			c.buf = nil
		}

		return nil, false
	}

	consumed := wire.EnvelopeSize + len(env.Body)
	c.buf = c.buf[consumed:]

	return env.Body, true
}
