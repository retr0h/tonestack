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
	"os"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
)

// debug traces every frame when TONESTACK_USB_DEBUG is set.
var debug = os.Getenv("TONESTACK_USB_DEBUG") != ""

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

	if debug {
		fmt.Fprintf(os.Stderr, "OUT %-8s seq=%d type=%#04x ack=%#x: %x\n",
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
func (s *session) receive(ctx context.Context, wait time.Duration) bool {
	rctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	buf := make([]byte, readBuffer)

	n, err := s.in.ReadContext(rctx, buf)
	if err != nil {
		// A read that fails is the device having nothing to say. That is what
		// a timeout looks like, and a timeout is the ordinary case: a device
		// is asked far more often than it answers.
		return false
	}

	if debug {
		fmt.Fprintf(os.Stderr, "IN  %d bytes: %x\n", n, buf[:min(n, 48)])
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

	return got
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
func (s *session) drain(ctx context.Context) {
	deadline := time.Now().Add(drainBudget)

	quiet := 0

	for quiet < drainQuietRuns && time.Now().Before(deadline) {
		// Somebody who stopped waiting is not owed a drained endpoint.
		if ctx.Err() != nil {
			break
		}

		if s.receive(ctx, drainReadWait) {
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
