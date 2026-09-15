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

package slots

import (
	"context"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// ImportWith puts a preset file into a slot on the given session.
//
// The preset is built into an unused slot the device itself wrote, so
// everything a chain does not describe is what the device expects to find
// there. See wire.Blank.
//
// The destination is overwritten and kept first. There is no undo on a device.
func (f *Flows) ImportWith(
	ctx context.Context,
	s device.Editor,
	file string,
	at slotpkg.Address,
) (result.Change, error) {
	doc, err := readPreset(ctx, file)
	if err != nil {
		return result.Change{}, err
	}

	body, err := f.documentFor(ctx, doc)
	if err != nil {
		return result.Change{}, err
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

	name := doc.Data.Meta.Name

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

	return out.Encode(), nil
}
