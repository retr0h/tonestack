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

package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// handshake opens every channel the editor uses.
//
// Once per session and never again. Answering a timeout by repeating this is
// the single most reliable way to wedge a device: every failure amplifies
// into a burst of them, and recovery needs the power supply pulled.
func (s *Session) handshake(ctx context.Context) error {
	s.drain(ctx)

	for _, spec := range channelSpecs {
		c := &channel{
			name: spec.name, device: spec.device, host: spec.host,
			txn: wire.FirstTxn,
		}
		s.chans[spec.name] = c

		for i, service := range spec.services {
			// A channel serving two services is opened twice, from scratch,
			// with the first closed in between. Multiplexing them onto one
			// open channel silently breaks every channel.
			if i > 0 {
				if err := s.closeChannel(c); err != nil {
					return err
				}

				// The device answers the close before it will answer a new
				// opening. Reopening without reading first leaves it talking
				// about the channel that just went away.
				s.receive(ctx, openReadWait)

				c.seq = 0
				c.rxBytes = 0
				c.buf = nil
			}

			if err := s.openService(ctx, c, service); err != nil {
				return err
			}
		}
	}

	return nil
}

// openService performs the three frames that bring one service up.
func (s *Session) openService(ctx context.Context, c *channel, service uint16) error {
	if err := s.send(c, wire.MsgHello, helloTail); err != nil {
		return err
	}

	// HX Edit's counter jumps from zero straight to two here, and the device
	// stops answering a client that sends one.
	c.seq = firstSeq

	// The device answers some openings and not others. A timeout here is not
	// a failure; the first request is what proves the session is alive.
	s.receive(ctx, openReadWait)

	body := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost, Service: service,
		Body: []byte{byte(service)},
	})

	if err := s.send(c, wire.MsgData, body); err != nil {
		return err
	}

	s.receive(ctx, openReadWait)

	return s.send(c, wire.MsgAck, nil)
}

// closeChannel sends the bare frame that releases a service.
//
// The device will not answer on a new service without it.
func (s *Session) closeChannel(c *channel) error {
	return s.send(c, wire.MsgHello, nil)
}

// Call makes one request and waits for its reply.
//
// The channel is acknowledged whenever bytes actually arrive. Without that
// the device stops feeding a reply partway through: it holds a window of
// about four kilobytes unacknowledged and then goes quiet, which looks
// exactly like a device that has stopped working.
func (s *Session) Call(
	ctx context.Context,
	channelName string,
	opcode uint64,
	args []wire.Arg,
) (wire.Response, error) {
	c, ok := s.chans[channelName]
	if !ok {
		return wire.Response{}, fmt.Errorf("no %s channel", channelName)
	}

	txn := c.txn
	c.txn++

	body := wire.EncodeRequest(wire.Request{Txn: txn, Opcode: opcode, Args: args})

	err := s.send(c, wire.MsgData, wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost, Service: 2, Body: body,
	}))
	if err != nil {
		return wire.Response{}, err
	}

	return s.awaitReply(ctx, c, txn, opcode)
}

// awaitReply reads until the reply to one transaction arrives.
func (s *Session) awaitReply(
	ctx context.Context,
	c *channel,
	txn, opcode uint64,
) (wire.Response, error) {
	deadline := time.Now().Add(replyBudget)

	for time.Now().Before(deadline) {
		// receive reports a failed read as the device having nothing to say,
		// which is right for a timeout and wrong for a cancelled context: a
		// read that returns instantly turns this into a spin, and the caller
		// is told the device never answered rather than that they stopped
		// waiting.
		if err := ctx.Err(); err != nil {
			return wire.Response{}, err
		}

		got := s.receive(ctx, replyReadWait)

		for {
			body, ok := message(c)
			if !ok {
				break
			}

			resp, err := wire.DecodeResponse(body)
			if err != nil {
				continue
			}

			// A notification carries no transaction and is not anybody's
			// reply. Letting one be mistaken for this reply would answer the
			// wrong question.
			if resp.Txn != txn {
				continue
			}

			if err := resp.Err(opcode); err != nil {
				return resp, err
			}

			return resp, nil
		}

		// Only when bytes actually arrived: the device sends empty transfers
		// when it has nothing to say, and acknowledging one burns a sequence
		// number and stalls the transfer.
		if got {
			if err := s.send(c, wire.MsgAck, nil); err != nil {
				return wire.Response{}, err
			}
		}
	}

	return wire.Response{}, fmt.Errorf(
		"no reply to opcode %d within %s", opcode, replyBudget)
}

// Presets lists what the device holds.
func (s *Session) Presets(ctx context.Context, setlist int) ([]wire.Preset, error) {
	resp, err := s.Call(ctx, channelControl, opListPresets, []wire.Arg{
		{Key: argSetlist, Value: uint64(setlist)},
		{Key: argListKind, Value: listKind},
	})
	if err != nil {
		return nil, err
	}

	return wire.DecodePresetList(resp.Result)
}

// ReadPreset fetches one slot without loading it.
//
// The device answers with its own document and goes on playing whatever it
// was. Nothing is selected and nothing is written.
//
// On the data channel, where every preset document goes, and carrying the
// same third argument a preset listing does. Both are needed: on the control
// channel, or without it, a device answers successfully with nothing at all.
//
// An empty slot answers with nothing too, which is a slot holding no preset
// rather than a failure: no bytes and no error.
func (s *Session) ReadPreset(
	ctx context.Context,
	setlist, slot int,
) ([]byte, error) {
	resp, err := s.Call(ctx, channelData, opReadPreset, []wire.Arg{
		{Key: argSetlist, Value: uint64(setlist)},
		{Key: argSlot, Value: uint64(slot)},
		{Key: argListKind, Value: listKind},
	})
	if err != nil {
		return nil, err
	}

	return document(resp.Result)
}
