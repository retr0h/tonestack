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

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Listing prints what a setlist holds.
//
// The operation that produced this decided nothing about how it looks: it
// handed back every slot and what is in each. Which of them to show, what
// colour a label is and how a chain reads as one line are decided here,
// because they are answers for a terminal and a terminal only.
func Listing(w io.Writer, l sdk.Listing, cat *catalog.Catalog, all bool) error {
	rows := make([][]string, 0, len(l.Slots))

	for _, h := range l.Slots {
		label := slotpkg.Label(h.Slot)

		if h.Empty() {
			// Shown greyed rather than left out, so somebody looking for
			// somewhere to put a preset can see where the gaps are.
			if all {
				rows = append(rows, []string{
					Mute(w, label), Mute(w, h.Name), Mute(w, "empty"),
				})
			}

			continue
		}

		rows = append(rows, []string{Accent(w, label), h.Name, Flow(w, h.Blocks, cat)})
	}

	return Section{
		Title:  l.Name,
		Detail: fmt.Sprintf("%s · %d in use", Plural(len(l.Slots), "slot"), l.Used()),
		// One address, the one printed on the pedal. What the device counts
		// underneath is its business, and --slot takes what is shown here.
		Headers: []string{"slot", "name", "chain"},
		Rows:    rows,
		Empty:   "no presets",
	}.Render(w)
}

// Plural renders a count with its noun, so a setlist of one does not read as
// "1 slots".
func Plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", n, noun)
}

// Flow reads a chain as one line of categories.
//
// Categories rather than model names, because a listing is for finding the
// preset you meant among a hundred and twenty-eight, and "drive → amp → cab"
// does that where four model names in a row do not.
func Flow(w io.Writer, blocks []chain.Block, cat *catalog.Catalog) string {
	parts := make([]string, 0, len(blocks))

	for _, b := range blocks {
		blk, ok := cat.Block(b.Model)
		if !ok {
			parts = append(parts, Info(w, "?"))

			continue
		}

		parts = append(parts, Category(w, blk.Category))
	}

	return strings.Join(parts, Mute(w, " → "))
}
