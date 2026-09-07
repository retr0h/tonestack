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

package resolve

import (
	"sort"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/corpus"
)

// nearUniversal is how common a kind of block must be before one is added to a
// chain that did not ask for it.
//
// Set high on purpose. A compressor in 88% of bass chains is a convention, and
// leaving it out produces something nobody would recognise. Drive in 61% is a
// choice, and making it silently would be this tool having opinions it cannot
// justify. A recipe naming a pedal always gets it, whatever the figure.
const nearUniversal = 0.75

// Added records a block the chain did not ask for, and why it is there.
//
// A generated rig is a set of decisions, and one made on the player's behalf
// has to be visible before they plug in rather than discovered after.
type Added struct {
	Block  catalog.Block
	Reason string
	Share  float64
}

// fill adds the blocks a chain of this kind almost always has.
//
// Only categories missing from the chain are considered, and only those the
// corpus shows to be near-universal for this instrument. What gets added is
// the model most people reach for, which is the only defensible choice when
// nobody named one.
func fill(
	blocks []catalog.Block,
	cat *catalog.Catalog,
	stats *corpus.Stats,
	instrument string,
) ([]catalog.Block, []Added) {
	if stats == nil {
		return blocks, nil
	}

	g, ok := stats.Grammar[instrument]
	if !ok {
		return blocks, nil
	}

	present := map[catalog.Category]bool{}
	for _, b := range blocks {
		present[b.Category] = true
	}

	var added []Added

	for _, c := range sortedByShare(g) {
		s := g.Categories[c]

		share := s.Frequency(g.Chains)
		if present[c] || share < nearUniversal {
			continue
		}

		pick, ok := commonest(cat, stats, c, instrument)
		if !ok {
			continue
		}

		blocks = insert(blocks, pick, s.BeforeAmp() >= 0.5)
		added = append(added, Added{
			Block: pick, Share: share,
			Reason: "almost every chain has one",
		})
	}

	return blocks, added
}

// insert places a block on the correct side of the amp.
//
// Position is not decoration: drive ahead of an amp overdrives its input,
// drive after it does something else entirely.
//
// A resolved chain always holds an amp — a recipe cannot omit one — so the
// index is found rather than guarded against.
func insert(
	blocks []catalog.Block,
	b catalog.Block,
	beforeAmp bool,
) []catalog.Block {
	if !beforeAmp {
		return append(blocks, b)
	}

	at := 0

	for i, existing := range blocks {
		if existing.Category == catalog.CategoryAmp {
			at = i

			break
		}
	}

	out := make([]catalog.Block, 0, len(blocks)+1)
	out = append(out, blocks[:at]...)
	out = append(out, b)

	return append(out, blocks[at:]...)
}

// commonest returns the model of a category that the corpus saw most often.
//
// Ties break on the identifier so a chain does not change between runs for
// reasons nobody chose.
func commonest(
	cat *catalog.Catalog,
	stats *corpus.Stats,
	c catalog.Category,
	instrument string,
) (catalog.Block, bool) {
	var (
		best catalog.Block
		uses int
		set  bool
	)

	for id, ms := range stats.Models {
		b, known := cat.Block(id)
		if !known || b.Category != c || catalog.NeedsUserIR(id) {
			continue
		}

		// A block whose cost was inferred rather than stated cannot be
		// budgeted honestly, and validation refuses one. Choosing it here
		// would produce a chain that is rejected a moment later, blaming a
		// block nobody asked for.
		if !b.DSP.Prov.Trusted() {
			continue
		}

		if !suits(b, instrument) {
			continue
		}

		if !set || ms.Uses > uses || (ms.Uses == uses && id < best.ID) {
			best, uses, set = b, ms.Uses, true
		}
	}

	return best, set
}

// suits reports whether a block is eligible for an instrument.
//
// Line 6 tags amps and cabinets Guitar or Bass; everything else is untagged
// and available to either.
func suits(b catalog.Block, instrument string) bool {
	if b.Subcategory == "" || !isInstrumentTag(b.Subcategory) {
		return true
	}

	return strings.EqualFold(b.Subcategory, instrument)
}

// sortedByShare orders categories by how often they appear, so the chain is
// filled with the most conventional blocks first and a budget runs out on the
// least important.
func sortedByShare(g corpus.Grammar) []catalog.Category {
	out := make([]catalog.Category, 0, len(g.Categories))
	for c := range g.Categories {
		out = append(out, c)
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := g.Categories[out[i]], g.Categories[out[j]]
		if a.Chains != b.Chains {
			return a.Chains > b.Chains
		}

		return out[i] < out[j]
	})

	return out
}
