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

// commitBudget is how long a device is given to finish a write.
//
// A write answers immediately to say it was accepted and reports finishing
// later. Treating the first answer as the end races the next write against a
// commit still running, which a device tolerates about a dozen times before
// it stops accepting writes.
var commitBudget = 10 * time.Second

// WritePreset puts a document into a slot.
//
// The document must be the bytes a device would have written. A preset is
// seeked through by a table of byte offsets, so one that differs in length
// from what it claims leaves those offsets pointing at the wrong places: the
// device accepts the write and then reads the preset as empty. Build it with
// wire.Document, which keeps a preset's own bytes and rebuilds the table.
//
// Waits for the device to say it finished, not merely that it accepted.
func (s *Session) WritePreset(
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
func (s *Session) WriteNamedPreset(
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
func (s *Session) write(
	ctx context.Context,
	opcode uint64,
	args []wire.Arg,
) error {
	c, ok := s.chans[channelData]
	if !ok {
		return fmt.Errorf("no %s channel", channelData)
	}

	txn := c.txn
	c.txn++

	body := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost,
		Service:    2,
		Body:       wire.EncodeRequest(wire.Request{Txn: txn, Opcode: opcode, Args: args}),
	})

	if err := s.stream(ctx, c, body); err != nil {
		return err
	}

	if _, err := s.awaitReply(ctx, c, txn, opcode); err != nil {
		return err
	}

	// A slot write answers and is done. The erase and program that follow do
	// not appear on the wire at all, and there is no completion notification
	// to wait for: a device that took the write and was then waited on
	// answers nothing for ten seconds while the preset it just wrote sits in
	// the slot. tonepush sends the same message as a plain request and sleeps
	// for the flash, which is what settle is.
	//
	// The status is not read for this reason. Both 0 and 1 have been seen for
	// a write that landed, and neither says anything about the erase.
	settle()

	return nil
}

// settle gives a write time to reach flash before the next one starts.
//
// A slot write answers as soon as the device has the document, and the erase
// and program that follow are not on the wire at all. A second write landing
// inside that window stacks its commit on the first. tonepush waits the same
// 750ms, and a restore writes preset after preset without it going wrong.
func settle() { time.Sleep(flashBudget) }

// flashBudget is how long that takes.
var flashBudget = 750 * time.Millisecond

// stream sends a message in the size a device takes, reading between frames
// so it can pace the sender.
func (s *Session) stream(ctx context.Context, c *channel, body []byte) error {
	for len(body) > 0 {
		n := min(len(body), streamChunk)

		if err := s.send(c, wire.MsgData, body[:n]); err != nil {
			return err
		}

		body = body[n:]

		// Between frames, not after the last one: a device that has more to
		// say says it now, and one with nothing to say costs a timeout.
		if len(body) > 0 {
			s.receive(ctx, replyReadWait)
		}
	}

	return nil
}
