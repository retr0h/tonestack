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
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// ShowOptions says which preset to show.
//
// A preset comes from either a slot in a setlist or a standalone .hlx file.
// Both read into the same rig, which is the point: what the device holds and
// what this tool generates are the same kind of thing.
type ShowOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// Path is the .hls or .hlb file to read. Empty when showing a file.
	Path string
	// File is a standalone .hlx to read. Empty when showing a slot.
	File string
	// Setlist selects one setlist within a bundle.
	Setlist int
	// Slot selects a position within the setlist.
	Slot int
	// CatalogPath is the generated catalog for the target device.
	CatalogPath string
}

// Show reads the rig a preset describes.
//
// A rig, not a rendering of one. RigSpec is what this project reads, writes
// and exchanges, so it is what looking at a preset produces — and what comes
// out here compiles back into the preset it came from, unchanged.
func Show(opts ShowOptions) (sdk.Reading, error) {
	doc, err := document(opts)
	if err != nil {
		return sdk.Reading{}, err
	}

	cat, err := opts.catalogs().Open(opts.CatalogPath)
	if err != nil {
		return sdk.Reading{}, err
	}

	// An empty slot is not a rig: it names no gear, and a rig holds at least
	// one thing. Answering with the name and nothing else beats an error
	// about a contract nobody broke.
	if c, err := doc.Spec(); err == nil && len(c.Blocks) == 0 {
		return sdk.Reading{Name: doc.Data.Meta.Name}, nil
	}

	spec, err := opts.compiler().Lift(doc, cat)
	if err != nil {
		return sdk.Reading{}, fmt.Errorf(
			"reading slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	return sdk.Reading{Name: doc.Data.Meta.Name, Doc: doc, Rig: spec}, nil
}

// document resolves the options to the preset they name.
func document(opts ShowOptions) (*preset.Document, error) {
	if opts.File != "" {
		return readFile(opts.File)
	}

	bundle, err := open(opts.Path)
	if err != nil {
		return nil, err
	}

	data, err := bundle.Slot(opts.Setlist, opts.Slot)
	if err != nil {
		return nil, err
	}

	return &preset.Document{
		Schema:  preset.Schema,
		Version: preset.Version,
		Data:    *data,
	}, nil
}

// readFile reads a standalone preset.
func readFile(path string) (*preset.Document, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	doc, err := preset.Read(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return doc, nil
}
