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

// ImportDevice puts a preset file into a slot on an attached device.
//
// The preset is built into an unused slot the device itself wrote, so
// everything a chain does not describe is what the device expects to find
// there. See wire.Blank.
//
// The destination is overwritten. There is no undo on a device.
func ImportDevice(ctx context.Context, opts ImportOptions) (result.Change, error) {
	s, err := OpenDevice(ctx)
	if err != nil {
		return result.Change{}, err
	}

	defer s.Close()

	return ImportWith(ctx, s, opts)
}

// ImportWith puts a preset file into a slot on the given session.
func ImportWith(
	ctx context.Context,
	s device.Editor,
	opts ImportOptions,
) (result.Change, error) {
	doc, err := readPreset(opts.File)
	if err != nil {
		return result.Change{}, err
	}

	body, err := documentFor(opts.Deps, doc, opts.CatalogPath)
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

	// What the slot holds now, before it stops holding it.
	kept, err := replacing(ctx, s, opts.Deps, opts.CatalogPath, opts.BackupDir,
		opts.Setlist, opts.Slot)
	if err != nil {
		return result.Change{}, err
	}

	name := doc.Data.Meta.Name

	if err := writer.WriteNamedPreset(
		ctx, opts.Setlist, opts.Slot, name, body); err != nil {
		return result.Change{}, fmt.Errorf(
			"writing slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	return result.Change{
		Action: result.Imported,
		To:     result.At{Slot: opts.Slot, Name: name},
		Kept:   kept,
	}, nil
}

// documentFor builds what a device holds out of what a file describes.
func documentFor(
	deps Deps,
	doc *preset.Document,
	catalogPath string,
) ([]byte, error) {
	cat, err := deps.catalogs().Open(catalogPath)
	if err != nil {
		return nil, err
	}

	blocks, err := deps.translator().Placements(doc, cat)
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
