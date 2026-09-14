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
	"errors"
	"fmt"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// send writes one frame on a channel and advances its counter.
//
// Every frame the host sends advances the sequence, acknowledgements and
// keep-alives included. The send lock is held for the whole of it, so frames
// leave in the order their numbers say.
func (s *session) send(
	c *channel,
	msgType uint16,
	payload []byte,
) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	return s.sendLocked(c, msgType, payload)
}

// sendLocked is send, for a caller already holding the send lock.
//
// The acknowledgement a frame carries is read as it goes out, so it covers
// every byte routed by then.
func (s *session) sendLocked(
	c *channel,
	msgType uint16,
	payload []byte,
) error {
	flags := wire.FlagNormal
	rx := c.rxBytes.Load()
	ack := wire.AckBase + rx
	hello := msgType == wire.MsgHello && payload != nil

	if hello {
		flags = wire.FlagHandshake
		ack = helloAck
	}

	raw := wire.EncodeFrame(wire.Frame{
		Flags: flags, DeviceNode: c.device, HostNode: c.host,
		Seq: c.seq, Type: msgType, Ack: ack, Payload: payload,
	})

	s.tracef("OUT %-8s seq=%d type=%#04x ack=%#x: %x\n",
		c.name, c.seq, msgType, ack, raw[:min(len(raw), 40)])

	if _, err := s.out.Write(raw); err != nil {
		return &busError{err: fmt.Errorf("writing to %s: %w", c.name, err)}
	}

	if !hello {
		c.ackSent.Store(rx)
	}

	c.seq++

	return nil
}

// nextTxn takes a channel's next transaction number.
func (s *session) nextTxn(
	c *channel,
) uint64 {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	txn := c.txn
	c.txn++

	return txn
}

// route takes one transfer apart and hands every frame in it to its channel.
//
// Holds the receive lock only while it appends, and never waits on a send,
// so a slow write cannot hold a read back.
func (s *session) route(
	transfer []byte,
) {
	s.tracef("IN  %d bytes: %x\n", len(transfer), transfer[:min(len(transfer), 48)])

	s.deliver(transfer)
	s.tick(true)
}

// deliver appends each frame's payload to its channel.
//
// A frame without stream bytes is not counted: the device sends empty
// transfers when it has nothing to say, and acknowledging one burns a
// sequence number and desynchronises the channel.
func (s *session) deliver(
	transfer []byte,
) {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	rest := transfer

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

		c.rxBytes.Add(uint32(len(f.Payload)))
		c.lastRx = time.Now()

		// Nothing reads the events channel, so its bytes are counted and
		// acknowledged, and kept nowhere.
		if c.name != channelEvents {
			c.buf = append(c.buf, f.Payload...)
		}

		select {
		case c.arrived <- struct{}{}:
		default:
		}
	}
}

// channelFor finds which open conversation a frame belongs to.
//
// The device swaps the node fields, so its frames carry the host node where
// a host frame carries the device node.
func (s *session) channelFor(
	f wire.Frame,
) *channel {
	for _, c := range s.chans {
		if c.open && (f.HostNode == c.device || f.DeviceNode == c.device) {
			return c
		}
	}

	return nil
}

// channel finds a channel the handshake opened.
func (s *session) channel(
	name string,
) (*channel, error) {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	c, ok := s.chans[name]
	if !ok || !c.open {
		return nil, fmt.Errorf("no %s channel", name)
	}

	return c, nil
}

// begin marks an exchange in flight on a channel, which keeps the idle
// acknowledgement off it. A session whose loop has ended starts nothing: no
// read is posted to catch the answer.
func (s *session) begin(
	c *channel,
) error {
	if err := s.ended(); err != nil {
		return err
	}

	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	c.busy = true

	return nil
}

// finish marks the exchange over.
func (s *session) finish(
	c *channel,
) {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	c.busy = false
}

// message takes one complete envelope out of a channel's buffer. The caller
// holds the receive lock.
//
// A reply is a byte stream split across frames at 256 bytes each, so it is
// complete only once the declared length has arrived.
func message(
	c *channel,
) ([]byte, bool) {
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
