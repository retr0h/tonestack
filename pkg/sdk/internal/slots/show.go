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
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Show reads the rig one slot of a file describes.
//
// A rig, not a rendering of one. RigSpec is what this project reads, writes and
// exchanges, so it is what looking at a preset produces, and what comes out
// here compiles back into the preset it came from, unchanged.
func (f *Flows) Show(
	ctx context.Context,
	path string,
	at slotpkg.Address,
) (result.Reading, error) {
	bundle, err := open(ctx, path)
	if err != nil {
		return result.Reading{}, err
	}

	data, err := bundle.Slot(at.Setlist, at.Slot)
	if err != nil {
		return result.Reading{}, err
	}

	return f.reading(ctx, &preset.Document{
		Schema:  preset.Schema,
		Version: preset.Version,
		Data:    *data,
	}, at.Slot)
}

// ShowFile reads the rig a standalone .hlx describes.
//
// The same rig a slot reads as, which is the point: what the device holds and
// what this tool generates are the same kind of thing.
func (f *Flows) ShowFile(
	ctx context.Context,
	file string,
) (result.Reading, error) {
	doc, err := readPreset(ctx, file)
	if err != nil {
		return result.Reading{}, err
	}

	// A file is one preset, so it is reported the way the first slot of a
	// setlist would be.
	return f.reading(ctx, doc, 0)
}

// reading lifts a preset into the rig it describes.
func (f *Flows) reading(
	ctx context.Context,
	doc *preset.Document,
	slot int,
) (result.Reading, error) {
	cat, err := f.catalog(ctx)
	if err != nil {
		return result.Reading{}, err
	}

	// An empty slot is not a rig: it names no gear, and a rig holds at least
	// one thing. Answering with the name and nothing else beats an error
	// about a contract nobody broke.
	if c, err := doc.Spec(); err == nil && len(c.Blocks) == 0 {
		return result.Reading{Name: doc.Data.Meta.Name}, nil
	}

	spec, err := f.compiler().Lift(doc, cat)
	if err != nil {
		return result.Reading{}, fmt.Errorf(
			"reading slot %s: %w", slotpkg.Label(slot), err)
	}

	return result.Reading{Name: doc.Data.Meta.Name, Doc: doc, Rig: spec}, nil
}

// readPreset reads a standalone preset.
func readPreset(
	ctx context.Context,
	path string,
) (*preset.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	doc, err := preset.Read(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return doc, nil
}
