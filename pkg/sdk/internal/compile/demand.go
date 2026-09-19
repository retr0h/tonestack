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
package compile

import (
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// needs is what a claim cannot mean anything without.
//
// Some of what a rig says names a block rather than a setting.
// `envelope-swept` does not ask for a filter to be turned up, it asserts
// there is one: a chain with no filter does not sweep however its knobs are
// set. Slap is the same shape of claim about how the instrument is played,
// and every source describing that sound describes compression with it.
//
// This is the half of the vocabulary that asks for a block. The other half
// moves controls, in turns, and nothing appears in both — a term that asked
// for a block and then set its knobs would be counted twice, once as a thing
// that exists and once as a thing that is turned up.
var needs = map[string]catalog.Category{
	"envelope-swept": catalog.CategoryFilter,
	"slap":           catalog.CategoryComp,
}

// demand adds the blocks the rig itself asks for, which fill cannot know
// about.
//
// fill answers "what does a chain of this kind usually have", from the corpus
// alone. That is the right question and it is not the only one. A rig also
// says things that need a particular block before they mean anything, and
// until this ran nothing carried that from the rig into the chain: the words
// were read after the blocks were chosen, so a word could only ever ask for a
// knob that already existed.
//
// Bootsy Collins is what this is for. `mid-forward` is the one claim in that
// rig earned by measuring records, and it moved nothing, because the
// amplifier standing in for his Alembic has no Mid and no equaliser was
// there to ask instead.
//
// Nothing here invents a value. It puts a block in the chain so that a word
// already earned has somewhere to land, and says which word asked for it.
func demand(
	blocks []catalog.Block,
	cat *catalog.Catalog,
	stats *corpus.Stats,
	spec rig.Spec,
	instrument string,
) ([]catalog.Block, []Added) {
	if stats == nil {
		return blocks, nil
	}

	g, known := stats.Grammar[instrument]
	if !known {
		return blocks, nil
	}

	var added []Added

	// Presence first. A word wanting a control may be answerable by a block
	// another claim has just asked for, and asking twice would seat two.
	for _, c := range claimed(spec) {
		want, names := needs[c]
		if !names || indexOf(blocks, want) >= 0 {
			continue
		}

		pick, ok := commonest(cat, stats, want, instrument, "")
		if !ok {
			continue
		}

		blocks = insert(blocks, pick, beforeAmp(g, want))
		added = append(added, Added{
			Block:  pick,
			Reason: fmt.Sprintf("the rig says %s, which needs one", c),
		})
	}

	terms := termsOf(spec)
	contested := contested(terms)

	for _, h := range terms {
		// Two words from one axis cancel rather than apply, and a word on
		// its way out must not seat a block on the way. The chain would
		// carry an equaliser that nothing then touched.
		if axis, ok := axisOf(h.term); ok && contested[axis] {
			continue
		}

		t, moves := turns[h.term]
		if !moves || t.absenceMeans != "" {
			continue
		}

		if _, _, found := answered(blocks, t); found {
			continue
		}

		pick, where, ok := somewhereFor(cat, stats, t, instrument)
		if !ok {
			continue
		}

		blocks = insert(blocks, pick, beforeAmp(g, pick.Category))
		added = append(added, Added{
			Block: pick,
			Reason: fmt.Sprintf(
				"the rig says %s and nothing here had a %s", h.term, where,
			),
		})
	}

	return blocks, added
}

// claimed is everything a rig says that might name a block.
//
// The character words and how the instrument is played, which are different
// fields saying the same kind of thing: not how loud, but what is there.
func claimed(
	spec rig.Spec,
) []string {
	out := []string{}

	for _, h := range termsOf(spec) {
		out = append(out, h.term)
	}

	if spec.Technique != nil {
		out = append(out, string(spec.Technique.Attack))
	}

	return out
}

// somewhereFor finds a block that could answer a term nothing in the chain
// can.
//
// The places are the term's own, in its own order: the block that usually
// answers it, then wherever else the same question can be put. An amplifier
// first, so a rig whose amp has a Mid is never given an equaliser it did not
// need, and the equaliser only when no amplifier here carries the band.
func somewhereFor(
	cat *catalog.Catalog,
	stats *corpus.Stats,
	t turn,
	instrument string,
) (catalog.Block, string, bool) {
	places := append([]answers{{t.category, t.param}}, t.also...)

	for _, place := range places {
		// An amplifier is never added to answer a word. A chain always has
		// one by the time this runs, and a second would be a different rig.
		if place.category == catalog.CategoryAmp {
			continue
		}

		pick, ok := commonest(cat, stats, place.category, instrument, place.param)
		if ok {
			return pick, place.param, true
		}
	}

	return catalog.Block{}, "", false
}

// beforeAmp says which side of the amplifier a demanded block belongs, from
// where the corpus puts that kind.
//
// The same question fill answers for the blocks it adds, asked the same way,
// so a block arriving because a word asked for it sits where one arriving
// because the corpus expected it would have.
//
// A kind the corpus has never counted goes after the amplifier. That is the
// safer end: a block ahead of an amplifier changes what the amplifier is fed,
// and guessing that from no evidence would be this deciding something nobody
// measured.
func beforeAmp(
	g corpus.Grammar,
	c catalog.Category,
) bool {
	s, seen := g.Categories[c]
	if !seen {
		return false
	}

	return s.BeforeAmp() >= 0.5
}
