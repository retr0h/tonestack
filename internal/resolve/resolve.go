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
// Package resolve turns curated knowledge into a signal chain.
//
// A recipe names real-world gear; a device understands model identifiers. This
// package is where the two meet: it looks up what each named piece of gear
// corresponds to on this device, orders the result into a chain the hardware
// can represent, and sets every parameter to what Line 6 says it should be.
//
// What it does not yet do is act on a recipe's character lines. Those describe
// how a rig should sound, and turning them into parameter moves is the next
// piece of work. Until then a generated preset is the right gear at factory
// settings, which is a starting point rather than an answer.
package resolve

import (
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/corpus"
	recipegen "github.com/retr0h/tonestack/pkg/recipe/gen"
)

// Resolve turns a recipe into a chain for the device the catalog describes.
func Resolve(
	rec *recipegen.Recipe,
	cat *catalog.Catalog,
	stats *corpus.Stats,
) (chain.Chain, []Added, error) {
	instrument := string(rec.InstrumentType)

	amp, err := findGear(cat, rec.Rig.Amp, catalog.CategoryAmp, instrument)
	if err != nil {
		return chain.Chain{}, nil, err
	}

	blocks := make([]catalog.Block, 0, 4)

	pedals, err := findPedals(cat, rec)
	if err != nil {
		return chain.Chain{}, nil, err
	}

	// Pedals reach the amp, the amp reaches the cabinet. That order is the
	// grammar of every guitar rig and is not a preference.
	blocks = append(blocks, pedals...)
	blocks = append(blocks, amp)

	if cab := findCab(cat, rec, amp); cab != nil {
		blocks = append(blocks, *cab)
	}

	blocks, added := fill(blocks, cat, stats, instrument)

	return spec(rec, blocks, stats), added, nil
}

// findPedals resolves each pedal the recipe names, in signal order.
func findPedals(cat *catalog.Catalog, rec *recipegen.Recipe) ([]catalog.Block, error) {
	if rec.Rig.Pedals == nil {
		return nil, nil
	}

	out := make([]catalog.Block, 0, len(*rec.Rig.Pedals))

	for _, name := range *rec.Rig.Pedals {
		// A pedal is not tagged by instrument, so the whole catalog is
		// eligible and any category will do — a recipe naming a delay is
		// naming a delay.
		b, err := findGear(cat, name, "", "")
		if err != nil {
			return nil, err
		}

		out = append(out, b)
	}

	return out, nil
}

// findCab resolves the cabinet, preferring what the recipe names.
//
// A recipe that names none gets the pairing Line 6 ships with the amp, which
// is a better answer than picking arbitrarily: it is what the model was voiced
// with.
func findCab(
	cat *catalog.Catalog,
	rec *recipegen.Recipe,
	amp catalog.Block,
) *catalog.Block {
	// Cabinets are not tagged by instrument the way amps are — Line 6 groups
	// them by routing — so the whole catalog is eligible.
	if rec.Rig.Cab != nil && *rec.Rig.Cab != "" {
		if b, err := findGear(cat, *rec.Rig.Cab, catalog.CategoryCab, ""); err == nil {
			return &b
		}

		// A cabinet the catalog does not name is not a reason to refuse to
		// build. Line 6 does not describe every cabinet in terms of real
		// gear, and the amp's own pairing is a better answer than nothing.
	}

	b, ok := cat.Block(amp.CabLink)
	if !ok {
		// An amp with no stated pairing, or one naming a cabinet this device
		// does not have. A chain without a cabinet is still a chain.
		return nil
	}

	return &b
}

// findGear returns the block emulating the named gear.
//
// Matching is on what Line 6 says a model is based on, because that is the
// only field naming gear a person recognises. An instrument narrows the search
// to the half of the catalog Line 6 tags that way, which is what keeps a bass
// request out of six hundred guitar models.
func findGear(
	cat *catalog.Catalog,
	gear string,
	category catalog.Category,
	instrument string,
) (catalog.Block, error) {
	want := strings.ToLower(gear)

	var best catalog.Block

	found := false

	for _, b := range cat.Blocks {
		if !eligible(b, want, category, instrument) {
			continue
		}

		if !found || closer(b, best) {
			best, found = b, true
		}
	}

	if !found {
		return catalog.Block{}, &NoSuchGearError{
			Gear: gear, Kind: kindOf(category), Instrument: instrument,
		}
	}

	return best, nil
}

// closer reports whether a is the better answer than b for the same query.
//
// Shorter wins: "Ampeg SVT" should find the SVT rather than the SVT-4 Pro, and
// a shorter description is the closer one. Where two are equally close the
// identifier decides, so the choice is the same every run — an ambiguous
// request such as "Ampeg SVT", which names neither the normal nor the bright
// channel, must not resolve differently because the catalog was regenerated.
//
// The chosen block is reported when a preset is built, so an ambiguity a
// person cares about is visible and can be settled by naming the channel in
// the recipe.
func closer(a, b catalog.Block) bool {
	if len(a.BasedOn) != len(b.BasedOn) {
		return len(a.BasedOn) < len(b.BasedOn)
	}

	return a.ID < b.ID
}

// eligible reports whether a block could be the gear being looked for.
func eligible(b catalog.Block, want string, category catalog.Category, instrument string) bool {
	if catalog.NeedsUserIR(b.ID) {
		return false
	}

	if !b.Matches(want) {
		return false
	}

	if category != "" && b.Category != category {
		return false
	}

	// Line 6 tags amps and cabinets Guitar or Bass; everything else is
	// untagged and available to either.
	if instrument != "" && b.Subcategory != "" && isInstrumentTag(b.Subcategory) {
		return strings.EqualFold(b.Subcategory, instrument)
	}

	return true
}

// isInstrumentTag reports whether a subcategory names an instrument rather
// than a routing shape such as "Mono, Stereo".
func isInstrumentTag(sub string) bool {
	return strings.EqualFold(sub, "guitar") || strings.EqualFold(sub, "bass")
}

// kindOf names a category for an error message.
func kindOf(c catalog.Category) string {
	if c == "" {
		return "block"
	}

	return string(c)
}

// spec lays blocks out as a chain the device can represent.
func spec(
	rec *recipegen.Recipe,
	blocks []catalog.Block,
	stats *corpus.Stats,
) chain.Chain {
	out := chain.Chain{
		Name:   rec.Name,
		Blocks: make([]chain.Block, 0, len(blocks)),
	}

	for i, b := range blocks {
		out.Blocks = append(out.Blocks, chain.Block{
			Model:   b.ID,
			Params:  settings(b, stats),
			DSP:     0,
			Pos:     i,
			Enabled: true,
		})
	}

	return out
}

// Fit reports whether a chain fits the device, moving blocks to the second
// processor when the first fills up.
//
// Line 6 states each block's cost as a percentage of one processor, so a chain
// that overflows is not a preset anyone can load.
func Fit(spec chain.Chain, cat *catalog.Catalog, lim chain.Limits) chain.Chain {
	used := 0.0

	for i := range spec.Blocks {
		b, ok := cat.Block(spec.Blocks[i].Model)
		if !ok {
			continue
		}

		cost := b.DSP.Mono
		if b.Stereo {
			cost = b.DSP.Stereo
		}

		if used+cost > lim.ChipCeiling {
			// A device with one signal path has nowhere to put the overflow.
			// Leaving the block where it is lets validation reject the chain,
			// which is the honest answer; moving it to a path the device does
			// not have would produce a file nothing can load.
			if lim.Paths > 1 {
				spec.Blocks[i].DSP = 1
			}

			continue
		}

		used += cost
	}

	return renumber(spec)
}

// renumber gives each processor a contiguous run of positions.
func renumber(spec chain.Chain) chain.Chain {
	next := map[int]int{}

	for i := range spec.Blocks {
		dsp := spec.Blocks[i].DSP
		spec.Blocks[i].Pos = next[dsp]
		next[dsp]++
	}

	return spec
}
