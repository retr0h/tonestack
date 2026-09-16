//go:build device

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

package device_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// opEmptySlot is opcode 16, which docs/protocol.md lists as "empty a slot" on
// the data channel taking a setlist and a slot.
//
// Nothing in this repository has ever sent it. The opcode, its channel and its
// arguments come from reading other people's implementations, and the status
// table in that document does not mark it verified. That is the whole reason
// this file exists.
const opEmptySlot = 16

// The argument keys opcode 16 takes: which setlist, and which slot in it. The
// package's own constants for these are unexported and this is an external
// test, so they are written out here rather than widened for an experiment.
const (
	keySetlist = 107
	keySlot    = 108
)

// experimentSetlist is the setlist the scratch slot is read and written in.
//
// Zero, which is the one an HX Stomp has and the one every other slot
// operation in this project addresses.
const experimentSetlist = 0

// flashPause is what a write waits for afterwards, and what this waits before
// reading the slot back.
//
// A slot write is finished when it is answered: the erase and the program that
// follow never appear on the wire, so nothing says when flash has settled
// except the clock. If emptying a slot erases flash the same way, reading it
// back at once could read the slot mid-erase. The same 750ms the write path
// uses, for the same reason.
const flashPause = 750 * time.Millisecond

// EmptySlotPublicTestSuite asks an attached HX Stomp what opcode 16 does.
//
// An experiment, not an assertion. Nobody has established that a device
// accepts this opcode at all, so the pedal refusing it is a result worth
// having rather than a failure: the test passes as long as it ran and put the
// slot back. What it observed is reported with t.Log.
//
// The question it settles is whether a swap with an empty slot can become a
// move. A swap is currently refused when either side is empty, because this
// project cannot produce what an empty slot holds and so will not move a
// preset and leave the source empty. If opcode 16 genuinely empties a slot,
// the refusal has an answer.
//
// The device build tag keeps it out of `just test` and continuous
// integration, and the slot it overwrites is named in the environment rather
// than chosen here.
type EmptySlotPublicTestSuite struct {
	suite.Suite
}

// TestEmptyASlot sends opcode 16 at the scratch slot and says what came back.
//
// In order: read the slot and keep both its document and its name, register
// putting it back, send the opcode, read the slot again and describe exactly
// which of the four cases it is. The restore is registered before anything is
// sent, so it runs whether or not the rest of the test survives.
//
// Raw through the session's own exchange path, which is the seam this package
// already exposes to its tests. Nothing in production gains a way to send an
// arbitrary opcode.
func (s *EmptySlotPublicTestSuite) TestEmptyASlot() {
	ctx := context.Background()
	t := s.T()

	// Emptying a slot on somebody's pedal is a choice they make, not a
	// default.
	label := os.Getenv("TONESTACK_SCRATCH_SLOT")
	s.Require().NotEmpty(label,
		"set TONESTACK_SCRATCH_SLOT to a slot this experiment may empty, such as 42C")

	var scratch int

	s.Require().NoError(slot.NewValue(&scratch).Set(label),
		"TONESTACK_SCRATCH_SLOT=%q is not a slot", label)

	// The same switch the CLI reads, so a hardware run leaves a frame trace.
	// This is the only record of what actually went over the wire, which is
	// how the two hardware-only findings in docs/protocol.md were made.
	var trace io.Writer
	if os.Getenv("TONESTACK_USB_DEBUG") != "" {
		trace = os.Stderr
	}

	editor, err := device.NewUSB(trace).Open(ctx)
	if errors.Is(err, device.ErrNoDevice) {
		t.Skip("no Helix attached")
	}

	s.Require().NoError(err)

	// Registered before anything that puts the slot back, so it runs after it:
	// cleanups run last first, and the restore needs the session open.
	t.Cleanup(func() {
		if err := editor.Close(); err != nil {
			t.Errorf("closing the session: %v", err)
		}
	})

	// Call is the raw exchange, and Editor deliberately does not carry it. The
	// concrete session is what this package hands its own tests.
	session, ok := editor.(*device.Session)
	s.Require().True(ok, "the opener returned %T, which has no raw call", editor)

	before, err := session.ReadPreset(ctx, experimentSetlist, scratch)
	s.Require().NoError(err)

	if len(before) == 0 {
		t.Skipf("%s already holds no preset, so emptying it would show nothing; "+
			"point TONESTACK_SCRATCH_SLOT at a slot holding something",
			slot.Label(scratch))
	}

	// The document carries no name: that lives in the listing, which is why
	// reading one slot costs two calls. Putting the slot back needs both.
	name := s.nameOf(ctx, session, scratch)

	t.Logf("%s holds %q, %d bytes, kept in memory for the restore",
		slot.Label(scratch), name, len(before))

	// Registered before the opcode goes out, so the slot is put back even if
	// sending it fails, if reading it back fails, or if the test panics.
	t.Cleanup(func() {
		restoreScratch(ctx, t, session, scratch, name, before)
	})

	resp, err := session.Call(ctx, device.DataChannel, opEmptySlot, []wire.Arg{
		wire.Number(keySetlist, experimentSetlist),
		wire.Number(keySlot, uint64(scratch)),
	})

	// Reported, never asserted. A refusal is an answer to the question.
	t.Logf("opcode %d on the data channel for %s answered status %d",
		opEmptySlot, slot.Label(scratch), resp.Status)

	if err != nil {
		t.Logf("and it came back as an error: %v", err)
	} else {
		t.Log("and it came back without an error")
	}

	// Whether or not it was accepted: if the device started an erase, reading
	// the slot back inside that window reads it mid-erase.
	time.Sleep(flashPause)

	after, readErr := session.ReadPreset(ctx, experimentSetlist, scratch)
	if readErr != nil {
		t.Logf("reading %s back failed: %v", slot.Label(scratch), readErr)

		return
	}

	t.Logf("after opcode %d, %s reads as: %s",
		opEmptySlot, slot.Label(scratch), describeSlot(before, after))
}

// nameOf reads what the device calls a slot.
//
// From the listing, because a document carries no name. Reported rather than
// required: a name that could not be read costs the restore its name and
// nothing else, and failing here would leave the experiment unrun.
func (s *EmptySlotPublicTestSuite) nameOf(
	ctx context.Context,
	session *device.Session,
	at int,
) string {
	presets, err := session.Presets(ctx, experimentSetlist)
	if err != nil {
		s.T().Logf("could not list presets to learn what %s is called: %v",
			slot.Label(at), err)

		return ""
	}

	if at >= len(presets) {
		s.T().Logf("the device listed %d slots, which does not reach %s",
			len(presets), slot.Label(at))

		return ""
	}

	return presets[at].Name
}

// describeSlot says which of the four cases a slot read back is.
//
// The cases are the ones worth telling apart. A slot that answers with no
// document at all is what a slot nobody has ever written answers with, and is
// the outcome that would turn the swap refusal into a move. A document with no
// blocks is a different thing: a whole preset that happens to hold no gear,
// which is what wire.Blank is and what a swap could already produce.
func describeSlot(
	before, after []byte,
) string {
	if len(after) == 0 {
		return "no document at all, which is how a slot that has never been " +
			"written answers"
	}

	if bytes.Equal(before, after) {
		return fmt.Sprintf("unchanged, the same %d bytes it held before", len(after))
	}

	preset, err := wire.DecodePreset(after)
	if err != nil {
		return fmt.Sprintf(
			"something else: %d bytes, where it held %d, and they do not decode "+
				"as a preset: %v", len(after), len(before), err)
	}

	if len(preset.Blocks) == 0 {
		return fmt.Sprintf(
			"a document with no blocks: %d bytes, decoding as a preset whose "+
				"chain is empty", len(after))
	}

	return fmt.Sprintf(
		"something else: %d bytes, where it held %d, decoding as a preset with "+
			"%d blocks", len(after), len(before), len(preset.Blocks))
}

// restoreScratch puts the kept copy back and says whether it landed.
//
// Reported rather than asserted, because it runs after the test body and a
// failure here has to say what state the pedal was left in. The copy is in
// memory only, so a restore that fails leaves the slot as the experiment left
// it and there is nothing on disk to recover it from: that is said plainly
// rather than implied.
//
// Named, because a write without one leaves whatever the slot was called,
// which after emptying it may be nothing.
func restoreScratch(
	ctx context.Context,
	t *testing.T,
	session *device.Session,
	at int,
	name string,
	kept []byte,
) {
	if err := session.WriteNamedPreset(ctx, experimentSetlist, at, name, kept); err != nil {
		t.Errorf("could not put %s back: %v; the pedal is left as the experiment "+
			"left it, and the only copy was in memory", slot.Label(at), err)

		return
	}

	back, err := session.ReadPreset(ctx, experimentSetlist, at)
	if err != nil {
		t.Errorf("could not read %s after putting it back: %v", slot.Label(at), err)

		return
	}

	if !bytes.Equal(kept, back) {
		t.Errorf("%s did not come back byte for byte: it now holds %d bytes "+
			"where it held %d", slot.Label(at), len(back), len(kept))

		return
	}

	t.Logf("%s put back as it was, %d bytes under %q", slot.Label(at), len(kept), name)
}

func TestEmptySlotPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EmptySlotPublicTestSuite))
}
