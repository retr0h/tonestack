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
	"io"
	"sort"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// ImportDevice puts a preset file into a slot on an attached device.
//
// The preset is built into an unused slot the device itself wrote, so
// everything a chain does not describe is what the device expects to find
// there. See wire.Blank.
//
// The destination is overwritten. There is no undo on a device.
func ImportDevice(ctx context.Context, w io.Writer, opts ImportOptions) error {
	s, err := openDevice(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	return ImportWith(ctx, w, s, opts)
}

// ImportWith puts a preset file into a slot on the given session.
func ImportWith(
	ctx context.Context,
	w io.Writer,
	s sdk.Editor,
	opts ImportOptions,
) error {
	doc, err := readPreset(opts.File)
	if err != nil {
		return err
	}

	body, err := documentFor(doc, opts.CatalogPath)
	if err != nil {
		return err
	}

	writer, err := writerFor(s)
	if err != nil {
		return err
	}

	name := doc.Data.Meta.Name

	if err := writer.WriteNamedPreset(
		ctx, opts.Setlist, opts.Slot, name, body); err != nil {
		return fmt.Errorf("writing slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	_, err = fmt.Fprintf(w, "\n%s%s %s %s\n\n%s%s\n\n",
		cli.Indent,
		cli.Accent(w, name),
		cli.Mute(w, "→"),
		cli.Accent(w, slotpkg.Label(opts.Slot)),
		cli.Indent, cli.Success(w, "written"))

	return err
}

// documentFor builds what a device holds out of what a file describes.
func documentFor(
	doc *preset.Document,
	catalogPath string,
) ([]byte, error) {
	cat, err := catalogview.Open(catalogPath)
	if err != nil {
		return nil, err
	}

	blocks, err := placementsOf(doc, cat)
	if err != nil {
		return nil, err
	}

	// An unused slot a device wrote, embedded in this binary and read by a
	// test, so neither of these can fail: it decodes, and it has a chain.
	out, _ := wire.Blank()
	open, _ := wire.Open(out)

	if err := pack(open, blocks); err != nil {
		return nil, err
	}

	if err := wire.Place(out, blocks); err != nil {
		return nil, err
	}

	return out.Encode(), nil
}

// pack puts a chain into the grid positions a device has free.
//
// A preset counts its blocks from zero along a signal path and a device
// counts positions across a grid that also holds the input, the split, the
// join and the output. The corpus puts every `@position` between 0 and 9,
// and a device's block positions run 1 to 8 and 11 to 18, so the two are not
// the same number and nothing here has established what turns one into the
// other.
//
// Rather than guess at that on somebody's hardware, the chain keeps its order
// and takes the first positions the device has free. A chain is an order, and
// the gaps a preset leaves in one carry no sound.
func pack(
	open []int,
	blocks []wire.Placement,
) error {
	if len(blocks) > len(open) {
		return fmt.Errorf(
			"this chain has %d blocks and the device lays out %d positions",
			len(blocks), len(open))
	}

	sort.SliceStable(blocks, func(i, j int) bool {
		return blocks[i].Position < blocks[j].Position
	})

	for i := range blocks {
		blocks[i].Position = open[i]
	}

	return nil
}
