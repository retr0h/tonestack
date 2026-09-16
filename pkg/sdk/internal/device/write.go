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

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// Opcodes that put a preset somewhere.
const (
	// opWritePreset writes a document into a slot, leaving its name alone.
	opWritePreset = 5
	// opWriteNamed writes a document and names it, which is what a paste or
	// an import does.
	opWriteNamed = 8
	// opEmptySlot takes away what a slot holds. It is the one operation
	// here that produces what a device answers for a slot nobody has
	// written, which nothing can build a document for.
	opEmptySlot = 16
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

// errUnacked is a chunk the device never acknowledged: the pace budget ran
// out waiting for its own channel's acknowledgement count to move, so
// nothing more of the message was sent.
var errUnacked = errors.New("the device did not acknowledge the chunk")

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

// EmptySlot takes away what a slot holds.
//
// Afterwards the slot answers with no document at all, which is what a slot
// nobody has ever written answers with. That is the one state this project
// cannot produce by writing, because there is no document to write for it,
// and it is what lets a swap with one empty side be a move.
//
// Sent on the data channel with the setlist and the slot, and nothing else.
// Verified on an HX Stomp on 15 September 2026: status 0, no error, and a
// slot holding a 2387-byte preset then read back as no document at all.
//
// Waits the flash pause afterwards, as a write does. The device answered it
// as a plain request, so nothing observed says a deferred commit follows; the
// pause is not for this call but for the next one. Whatever an empty does to
// flash is not on the wire either, and a write landing straight after it
// would stack on top of it, which is the case the pause exists for.
func (s *session) EmptySlot(
	ctx context.Context,
	setlist, slot int,
) error {
	return s.write(ctx, opEmptySlot, []wire.Arg{
		wire.Number(argSetlist, uint64(setlist)),
		wire.Number(argSlot, uint64(slot)),
	})
}

// write sends one request too large for a single frame, and waits for the
// device to finish acting on it.
func (s *session) write(
	ctx context.Context,
	opcode uint64,
	args []wire.Arg,
) error {
	// Before anything is sent. Afterwards a caller who stops waiting does not
	// stop it: only the device does, by taking each chunk or by going quiet
	// long enough that pace ends the session, so this call's tail never
	// overlaps the next one's head.
	if err := ctx.Err(); err != nil {
		return err
	}

	c, err := s.channel(channelData)
	if err != nil {
		return err
	}

	// In flight from the first chunk through the flash pause, so the idle
	// acknowledgement sends nothing on any channel inside the window a device
	// punishes.
	if err := s.begin(); err != nil {
		return err
	}

	defer s.finish()

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
	txn := s.nextTxn(c)

	body := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost,
		Service:    2,
		Body:       wire.EncodeRequest(wire.Request{Txn: txn, Opcode: opcode, Args: args}),
	})

	// A write that has started finishes, and its answer is read, whoever
	// stops waiting; the next operation is the one that sees the
	// cancellation. Between chunks the message is bounded regardless: the
	// pace budget is what stops it if the device goes quiet, and running out
	// ends the session rather than leave the data channel holding half a
	// message for a later call to feed a fresh request into.
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
	<-s.after(s.budgets.flash)
}

// stream sends a message in the size a device takes, waiting for the
// device's own acknowledgement between frames so it can pace the sender.
//
// Between frames, not after the last one: a device that has more to say says
// it now, and one with nothing to say costs the pace budget. No chunk goes
// out without the device's acknowledgement of the one before it: a hardware
// trace showed a chunk released by a frame on another channel, not its own
// acknowledgement, and every chunk sent after it stalled the endpoint. So a
// pace that cannot get one stops the message where it is, rather than
// sending the rest blind.
func (s *session) stream(
	c *channel,
	body []byte,
) error {
	total := len(body)
	sent := 0

	for len(body) > 0 {
		n := min(len(body), streamChunk)
		mark := c.acked()

		if err := s.send(c, wire.MsgData, body[:n]); err != nil {
			return err
		}

		sent += n
		body = body[n:]

		if len(body) > 0 {
			if err := s.pace(c, mark); err != nil {
				return fmt.Errorf(
					"the device stopped taking the message after %d of %d bytes: %w",
					sent, total, err)
			}
		}
	}

	return nil
}
