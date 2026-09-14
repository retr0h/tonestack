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

// handshake opens every channel the editor uses.
//
// Once per session and never again. Answering a timeout by repeating this is
// the single most reliable way to wedge a device: every failure amplifies
// into a burst of them, and recovery needs the power supply pulled.
func (s *session) handshake(
	ctx context.Context,
) error {
	s.drain(ctx)

	// A bus that failed during the drain is not one to open channels on.
	if err := s.ended(); err != nil {
		return err
	}

	for _, spec := range channelSpecs {
		if err := s.openChannel(ctx, s.chans[spec.name], spec.services); err != nil {
			return err
		}
	}

	return nil
}

// openChannel brings up every service a channel serves.
//
// The channel is busy for the length of it, so the idle acknowledgement stays
// out of an opening.
func (s *session) openChannel(
	ctx context.Context,
	c *channel,
	services []uint16,
) error {
	s.rxMu.Lock()
	c.open, c.busy = true, true
	s.rxMu.Unlock()

	defer s.finish(c)

	for i, service := range services {
		// A channel serving two services is opened twice, from scratch, with
		// the first closed in between. Multiplexing them onto one open channel
		// silently breaks every channel.
		if i > 0 {
			mark := s.progress().transfers

			if err := s.closeChannel(c); err != nil {
				return err
			}

			// The device answers the close before it will answer a new
			// opening. Reopening without waiting first leaves it talking about
			// the channel that just went away.
			if err := s.pause(ctx, mark, s.budgets.open); err != nil {
				return err
			}

			s.reopen(c)
		}

		if err := s.openService(ctx, c, service); err != nil {
			return err
		}
	}

	return nil
}

// reopen puts a channel's counters back where a new opening starts them.
func (s *session) reopen(
	c *channel,
) {
	s.sendMu.Lock()
	c.seq = 0
	s.sendMu.Unlock()

	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	c.rxBytes.Store(0)
	c.ackSent.Store(0)
	c.buf = nil
}

// openService performs the three frames that bring one service up.
func (s *session) openService(
	ctx context.Context,
	c *channel,
	service uint16,
) error {
	mark := s.progress().transfers

	if err := s.send(c, wire.MsgHello, helloTail); err != nil {
		return err
	}

	// HX Edit's counter jumps from zero straight to two here, and the device
	// stops answering a client that sends one.
	s.sendMu.Lock()
	c.seq = firstSeq
	s.sendMu.Unlock()

	// The device answers some openings and not others. Silence here is not a
	// failure; the first request is what proves the session is alive. A bus
	// that fails is, and a handshake is not retried.
	if err := s.pause(ctx, mark, s.budgets.open); err != nil {
		return err
	}

	body := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost, Service: service,
		Body: []byte{byte(service)},
	})

	mark = s.progress().transfers

	if err := s.send(c, wire.MsgData, body); err != nil {
		return err
	}

	if err := s.pause(ctx, mark, s.budgets.open); err != nil {
		return err
	}

	return s.send(c, wire.MsgAck, nil)
}

// closeChannel sends the bare frame that releases a service.
//
// The device will not answer on a new service without it.
func (s *session) closeChannel(
	c *channel,
) error {
	return s.send(c, wire.MsgHello, nil)
}

// Call makes one request and waits for its reply.
//
// The channel is acknowledged whenever bytes actually arrive. Without that
// the device stops feeding a reply partway through: it holds a window of
// about four kilobytes unacknowledged and then goes quiet, which looks
// exactly like a device that has stopped working.
func (s *session) Call(
	ctx context.Context,
	channelName string,
	opcode uint64,
	args []wire.Arg,
) (wire.Response, error) {
	c, err := s.channel(channelName)
	if err != nil {
		return wire.Response{}, err
	}

	if err := s.begin(c); err != nil {
		return wire.Response{}, err
	}

	defer s.finish(c)

	txn := s.nextTxn(c)
	body := wire.EncodeRequest(wire.Request{Txn: txn, Opcode: opcode, Args: args})

	err = s.send(c, wire.MsgData, wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost, Service: 2, Body: body,
	}))
	if err != nil {
		return wire.Response{}, err
	}

	return s.awaitReply(ctx, c, txn, opcode, s.budgets.reply)
}

// awaitReply waits for the reply to one transaction, or for budget to run
// out.
//
// The budget is the caller's: a call is answered within the reply budget, and
// a write, which answers once it has committed, is given the commit budget.
func (s *session) awaitReply(
	ctx context.Context,
	c *channel,
	txn, opcode uint64,
	budget time.Duration,
) (wire.Response, error) {
	timer := time.NewTimer(budget)
	defer timer.Stop()

	for {
		// Before the buffer, so somebody who stopped waiting is told that
		// rather than handed an answer they no longer want.
		if err := ctx.Err(); err != nil {
			return wire.Response{}, err
		}

		if resp, ok, err := s.reply(c, txn, opcode); ok {
			return resp, err
		}

		// After the buffer, so an answer routed just before the bus failed is
		// still read.
		if err := s.ended(); err != nil {
			return wire.Response{}, err
		}

		// Only when bytes actually arrived: the device sends empty transfers
		// when it has nothing to say, and acknowledging one burns a sequence
		// number and stalls the transfer.
		if c.owed() {
			if err := s.send(c, wire.MsgAck, nil); err != nil {
				return wire.Response{}, err
			}
		}

		select {
		case <-c.arrived:
		case <-s.dead:
		case <-ctx.Done():
		case <-timer.C:
			return wire.Response{}, fmt.Errorf(
				"%w to opcode %d within %s", errNoReply, opcode, budget)
		}
	}
}

// reply takes envelopes off a channel's buffer until one answers txn.
//
// Reports whether it found one, and what that answer says.
func (s *session) reply(
	c *channel,
	txn, opcode uint64,
) (wire.Response, bool, error) {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	for body, ok := message(c); ok; body, ok = message(c) {
		resp, err := wire.DecodeResponse(body)
		if err != nil {
			continue
		}

		// A notification carries no transaction and is not anybody's reply.
		// Letting one be mistaken for this reply would answer the wrong
		// question. A late reply to a call somebody gave up on is skipped the
		// same way.
		if resp.Txn != txn {
			continue
		}

		return resp, true, resp.Err(opcode)
	}

	return wire.Response{}, false, nil
}

// errNoReply is a device that stayed silent for the whole of its budget.
//
// Told apart from a bus that failed, because a device busy switching presets
// goes quiet too, and that is worth asking again.
var errNoReply = errors.New("no reply")

// Presets lists what the device holds.
func (s *session) Presets(
	ctx context.Context,
	setlist int,
) ([]wire.Preset, error) {
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
func (s *session) ReadPreset(
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
