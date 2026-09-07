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
	"io"
	"os"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
)

// ShowOptions says which preset to show.
//
// A preset comes from either a slot in a setlist or a standalone .hlx file.
// Both decode to the same chain, which is the point: what the device holds
// and what this tool generates are the same kind of thing.
type ShowOptions struct {
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

// Show prints the chain a preset describes.
func Show(w io.Writer, opts ShowOptions) error {
	spec, where, err := load(opts)
	if err != nil {
		return err
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	if err := header(w, spec.Name, where); err != nil {
		return err
	}

	if err := cli.Chain(w, spec, cat); err != nil {
		return err
	}

	_, err = fmt.Fprintln(w)

	return err
}

// load resolves the options to a chain, and to a description of where it came
// from for the header to show.
func load(opts ShowOptions) (chain.Chain, string, error) {
	if opts.File != "" {
		spec, err := readFile(opts.File)

		return spec, opts.File, err
	}

	doc, err := open(opts.Path)
	if err != nil {
		return chain.Chain{}, "", err
	}

	data, err := doc.Slot(opts.Setlist, opts.Slot)
	if err != nil {
		return chain.Chain{}, "", err
	}

	spec, err := data.Spec()
	if err != nil {
		return chain.Chain{}, "", fmt.Errorf("reading slot %d: %w", opts.Slot, err)
	}

	return spec, "slot " + position(opts.Slot), nil
}

// readFile reads a chain out of a standalone preset.
func readFile(path string) (chain.Chain, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return chain.Chain{}, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	doc, err := preset.Read(f)
	if err != nil {
		return chain.Chain{}, fmt.Errorf("reading %s: %w", path, err)
	}

	return doc.Spec()
}
