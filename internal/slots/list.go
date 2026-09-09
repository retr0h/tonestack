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
	"strings"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/setlist"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// ListOptions says which setlist to list.
type ListOptions struct {
	// Path is the .hls or .hlb file to read.
	Path string
	// Setlist selects one setlist within a bundle.
	Setlist int
	// CatalogPath is the generated catalog, used to summarise each chain.
	CatalogPath string
	// All includes slots holding no blocks.
	All bool
}

// List prints every slot in a setlist.
//
// Empty slots are hidden by default. A device-written setlist always holds
// 128 of them and most are untouched, so listing them all buries the ones
// somebody actually made.
func List(w io.Writer, opts ListOptions) error {
	doc, err := open(opts.Path)
	if err != nil {
		return err
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	if opts.Setlist < 0 || opts.Setlist >= len(doc.Setlists) {
		return &setlist.NoSuchSlotError{Setlist: opts.Setlist}
	}

	sl := doc.Setlists[opts.Setlist]
	rows, used := listRows(w, sl, cat, opts.All)

	return cli.Section{
		Title:   sl.Name(),
		Detail:  fmt.Sprintf("%s · %d in use", plural(len(sl.Slots), "slot"), used),
		Headers: []string{"slot", "name", "chain"},
		Rows:    rows,
		Empty:   "no presets",
	}.Render(w)
}

// listRows renders one row per slot, and counts the ones holding a chain.
func listRows(
	w io.Writer,
	sl setlist.Setlist,
	cat *catalog.Catalog,
	all bool,
) ([][]string, int) {
	var (
		rows [][]string
		used int
	)

	for i := range sl.Slots {
		// A slot that fails to parse is still a slot; showing it as empty is
		// better than refusing to list the 127 around it.
		spec, _ := sl.Slots[i].Spec()

		if len(spec.Blocks) == 0 {
			if all {
				rows = append(rows, []string{
					cli.Mute(w, slotpkg.Label(i)),
					cli.Mute(w, sl.Slots[i].Meta.Name),
					"",
				})
			}

			continue
		}

		used++

		rows = append(rows, []string{
			cli.Accent(w, slotpkg.Label(i)),
			sl.Slots[i].Meta.Name,
			flow(w, spec.Blocks, cat),
		})
	}

	return rows, used
}

// flow summarises a chain as the categories it passes through.
//
// The categories are what distinguishes one preset from another at a glance —
// whether it has an amp, whether anything comes after the cab. Model names
// would be more precise and would not fit on a line.
func flow(w io.Writer, blocks []chain.Block, cat *catalog.Catalog) string {
	parts := make([]string, 0, len(blocks))

	for _, b := range blocks {
		blk, ok := cat.Block(b.Model)
		if !ok {
			parts = append(parts, cli.Info(w, "?"))

			continue
		}

		parts = append(parts, cli.Category(w, blk.Category))
	}

	return strings.Join(parts, cli.Mute(w, " → "))
}

// plural renders a count with its noun, so a setlist of one does not read as
// "1 slots".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", n, noun)
}
