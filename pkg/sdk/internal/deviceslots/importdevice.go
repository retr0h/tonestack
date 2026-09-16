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

package deviceslots

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// rawExt is what a backup of a slot nothing could read a chain out of is
// called. The bytes in it are the device's own, so putting one back writes
// them as they are. See the backup package's rules 3 and 4.
const rawExt = ".bin"

// Import puts a preset file into a slot on the given session.
//
// The preset is built into an unused slot the device itself wrote, so
// everything a chain does not describe is what the device expects to find
// there. See wire.Blank.
//
// The destination is overwritten and kept first. There is no undo on a device.
func (f *Flows) Import(
	ctx context.Context,
	s device.Editor,
	file string,
	at slotpkg.Address,
) (result.Change, error) {
	// A .bin is what a backup holds for a slot nothing could read a chain
	// out of: the device's own bytes, kept exactly as they arrived. There is
	// no document in it to rebuild from and no name inside it, so it goes
	// back the way it came off, and the slot keeps whatever it is called.
	raw := filepath.Ext(file) == rawExt

	var (
		body []byte
		doc  *preset.Document
		err  error
	)

	if raw {
		body, err = os.ReadFile(file) //nolint:gosec // the path is the user's own file
		if err != nil {
			return result.Change{}, fmt.Errorf("reading %s: %w", file, err)
		}
	} else {
		doc, err = fileslots.ReadPreset(ctx, file)
		if err != nil {
			return result.Change{}, err
		}

		body, err = f.documentFor(ctx, doc)
		if err != nil {
			return result.Change{}, err
		}
	}

	// Before the backup rather than after it: a session that cannot write
	// is not going to replace anything, so reading the slot to keep it would
	// be a round trip to the device for nothing.
	writer, err := writerFor(s)
	if err != nil {
		return result.Change{}, err
	}

	// What the device calls the slot, so the copy kept of it carries that
	// name. The preset read back for one slot does not say.
	found, err := s.Presets(ctx, at.Setlist)
	if err != nil {
		return result.Change{}, fmt.Errorf("listing presets: %w", err)
	}

	// What the slot holds now, before it stops holding it.
	kept, err := f.replacing(ctx, s, at, nameOf(found, at.Slot))
	if err != nil {
		return result.Change{}, err
	}

	// A .bin carries no name of its own, so the slot keeps the one it has.
	name := nameOf(found, at.Slot)
	if !raw {
		name = doc.Data.Meta.Name
	}

	if err := writer.WriteNamedPreset(ctx, at.Setlist, at.Slot, name, body); err != nil {
		return result.Change{}, keptError(fmt.Errorf(
			"writing slot %s: %w", slotpkg.Label(at.Slot), err), kept)
	}

	return result.Change{
		Action: result.Imported,
		To:     result.At{Slot: at.Slot, Name: name},
		Kept:   kept,
	}, nil
}

// documentFor builds what a device holds out of what a file describes.
func (f *Flows) documentFor(
	ctx context.Context,
	doc *preset.Document,
) ([]byte, error) {
	cat, err := f.catalog(ctx)
	if err != nil {
		return nil, err
	}

	blocks, err := f.translator().Placements(doc, cat)
	if err != nil {
		return nil, err
	}

	// An unused slot a device wrote, embedded in this binary and read by a
	// test, so it cannot fail to decode.
	out, _ := wire.Blank()

	if err := wire.PlaceAsWritten(out, blocks); err != nil {
		return nil, err
	}

	// After the chain. Placing it writes every snapshot's record of what is
	// switched on from the chain itself, which is right for a preset carrying
	// no snapshots of its own and wrong for one that does: it would leave
	// three snapshots recalling the same sound under the blank's names.
	wire.PlaceSnapshots(out, f.translator().SnapshotStates(doc))

	// And the routing, for the same reason. A chain is written into the
	// sixteen positions a device gives it and never into the four its input,
	// split, join and output sit on, so without this a preset keeps the
	// routing of the slot it was built into rather than the one the file
	// describes.
	// The error is a value that will not encode, and everything reaching here
	// came out of rawValue, which narrows each one to a bool, an integer or a
	// float. PlaceRouting keeps reporting it for callers that build their own.
	_ = wire.PlaceRouting(
		out, f.translator().RoutingStates(doc, cat, wire.BlankRouting()))

	return out.Encode(), nil
}
