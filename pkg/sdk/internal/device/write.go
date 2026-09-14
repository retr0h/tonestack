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

// Opcodes that put a preset somewhere.
const (
	// opWritePreset writes a document into a slot, leaving its name alone.
	opWritePreset = 5
	// opWriteNamed writes a document and names it, which is what a paste or
	// an import does.
	opWriteNamed = 8
)

// Argument keys a write uses beyond the ones a read does.
//
// HX Edit's own traffic carries three more, 123, 124 and 125, holding false,
// false and 0. They are not sent here. tonepush's implementation omits them
// and its writes land, so they are something HX Edit says rather than
// something a device needs.
const (
	argName     = 109
	argDocument = 110
)

// streamChunk is how much of a message a device takes per frame.
//
// It paces the sender with acknowledgements. Writing a whole preset at once
// fills its receive window and stalls the endpoint: the transfer times out,
// and the interface will not be claimed again until the device is power
// cycled.
const streamChunk = 256

// WritePreset puts a document into a slot.
//
// The document must be the bytes a device would have written. A preset is
// seeked through by a table of byte offsets, so one that differs in length
// from what it claims leaves those offsets pointing at the wrong places: the
// device accepts the write and then reads the preset as empty. Build it with
// wire.Document, which keeps a preset's own bytes and rebuilds the table.
//
// Waits for the device to say it finished, not merely that it accepted.
func (s *session) WritePreset(
	ctx context.Context,
	setlist, slot int,
	document []byte,
) error {
	return s.write(ctx, opWritePreset, []wire.Arg{
		wire.Number(argSetlist, uint64(setlist)),
		wire.Number(argSlot, uint64(slot)),
		wire.Blob(argDocument, document),
	})
}

// WriteNamedPreset puts a document into a slot under a name.
//
// What a paste or an import does. Writing without the name leaves whatever
// the slot was called, which is right for editing a preset in place and wrong
// for putting a different one there.
func (s *session) WriteNamedPreset(
	ctx context.Context,
	setlist, slot int,
	name string,
	document []byte,
) error {
	return s.write(ctx, opWriteNamed, []wire.Arg{
		wire.Number(argSetlist, uint64(setlist)),
		wire.Number(argSlot, uint64(slot)),
		wire.Text(argName, name),
		wire.Blob(argDocument, document),
	})
}

// write sends one request too large for a single frame, and waits for the
// device to finish acting on it.
func (s *session) write(
	ctx context.Context,
	opcode uint64,
	args []wire.Arg,
) error {
	// Before anything is sent. Afterwards the message is finished whatever
	// happens: a device fed half a message and then a burst is the stall
	// docs/protocol.md describes.
	if err := ctx.Err(); err != nil {
		return err
	}

	c, err := s.channel(channelData)
	if err != nil {
		return err
	}

	if err := s.commit(ctx, c, opcode, args); err != nil {
		return err
	}

	// A slot write answers and is done. The erase and program that follow do
	// not appear on the wire at all, and there is no completion notification
	// to wait for: a device that took the write and was then waited on
	// answers nothing for ten seconds while the preset it just wrote sits in
	// the slot. tonepush sends the same message as a plain request and sleeps
	// for the flash, which is what settle is.
	s.settle()

	return nil
}

// commit is the exchange a write is: the message, then its answer.
//
// Both 0 and 1 have been seen for a write that landed, and neither says
// anything about the erase, so either is success. Any other status fails in
// awaitReply, which reads it through wire.Response.Err: 255 is a refusal, and
// anything else is a status nobody has seen.
func (s *session) commit(
	ctx context.Context,
	c *channel,
	opcode uint64,
	args []wire.Arg,
) error {
	if err := s.begin(c); err != nil {
		return err
	}

	defer s.finish(c)

	txn := s.nextTxn(c)

	body := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost,
		Service:    2,
		Body:       wire.EncodeRequest(wire.Request{Txn: txn, Opcode: opcode, Args: args}),
	})

	// A write that has started finishes, and its answer is read, whoever
	// stops waiting; the next operation is the one that sees the
	// cancellation. The message itself carries no deadline at all: a budget
	// that ran out between chunks would leave the device holding half of it.
	// It is finite regardless, one bounded pause per chunk.
	if err := s.stream(c, body); err != nil {
		return err
	}

	// The commit budget bounds the wait for the answer, and only that. A call
	// gets the shorter reply budget; a write answers once it has committed,
	// and giving up sooner races the next write against a commit still
	// running.
	if _, err := s.awaitReply(
		context.WithoutCancel(ctx), c, txn, opcode, s.budgets.commit); err != nil {
		// Detached from the caller, so silence here is the budget running out.
		if errors.Is(err, errNoReply) {
			return fmt.Errorf("%w, the commit budget: %w", err, context.DeadlineExceeded)
		}

		return err
	}

	return nil
}

// settle gives a write time to reach flash before the next one starts.
//
// A slot write answers as soon as the device has the document, and the erase
// and program that follow are not on the wire at all. A second write landing
// inside that window stacks its commit on the first. tonepush waits the same
// 750ms, and a restore writes preset after preset without it going wrong.
//
// A wait on a timer, not a sleep of the only reader: the loop goes on reading
// throughout.
func (s *session) settle() {
	timer := time.NewTimer(s.budgets.flash)
	defer timer.Stop()

	<-timer.C
}

// stream sends a message in the size a device takes, pausing between frames
// so it can pace the sender.
//
// Only a failed send stops it. Between frames, not after the last one: a
// device that has more to say says it now, and one with nothing to say costs
// the pace budget. A loop that has ended does not stop it either, because a
// message that has started must go out whole: a device left holding half of
// one is the stall docs/protocol.md describes. A bus that has really gone
// fails the next send, which does stop the message.
func (s *session) stream(
	c *channel,
	body []byte,
) error {
	for len(body) > 0 {
		n := min(len(body), streamChunk)
		mark := s.progress().transfers

		if err := s.send(c, wire.MsgData, body[:n]); err != nil {
			return err
		}

		body = body[n:]

		if len(body) > 0 {
			s.pace(mark)
		}
	}

	return nil
}
