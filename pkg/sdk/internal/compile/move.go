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
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Moved is what a character term did to a parameter.
type Moved struct {
	// Term is the word that moved it.
	Term string
	// Param is the control it moved. Empty when nothing in the chain answers
	// to this term yet, which is most of the vocabulary.
	Param string
	// From and To are where the parameter was and where it went.
	From float64
	To   float64
	// Against names the axis another term in the same rig also spoke for.
	// Empty unless two words answered one question.
	Against string
	// Because says why the chain could not answer this word, when the chain
	// is the reason. Empty when nothing acts on the word at all, which is a
	// different answer: one says this rig cannot hear it, the other says
	// nobody has taught the project to listen.
	Because string
	// Already says how the chain answers this word without a knob being
	// turned. A rig asking for no room, in a chain holding no reverb, asked
	// for something it already has.
	Already string
}

// Acted says whether the term moved anything.
func (m Moved) Acted() bool { return m.Param != "" }

// Contested says whether another term spoke for the same axis.
func (m Moved) Contested() bool { return m.Against != "" }

// Unanswered says whether the chain, rather than this project, is why the
// word moved nothing.
func (m Moved) Unanswered() bool { return m.Because != "" }

// Holds says whether the chain already answers the word as built.
func (m Moved) Holds() bool { return m.Already != "" }

// turn is one term's effect: which kind of block, which parameter, and how
// many steps along it.
//
// A step is signed and scaled rather than a direction, because two shapes of
// axis need different arithmetic. `mids` is a pair, one step either side.
// `drive` is a scale — clean, minimal-drive, grit-on-attack, saturated are
// four positions on one line — so each term carries its own multiple.
type turn struct {
	category catalog.Category
	param    string
	steps    float64
	// absenceMeans is what it means for the chain to hold no block of this
	// kind, where that already answers the term. Empty where it does not:
	// a chain with no compressor is not a chain with a soft attack, but a
	// chain with no reverb really does have no room on it.
	absenceMeans string
}

// turns is what a term does, for the terms that do anything.
//
// Four axes of the ten, because four have a direction that is not a guess.
// Mid, Treble and Drive need no explaining. Sag does, and the Pilot's Guide
// explains it: lower values offer tighter responsiveness, higher values more
// touch dynamics and sustain.
//
// The other six axes — decay, attack, space, string-noise, pickup, movement —
// are not amplifier controls. A term from one of those is recorded and moves
// nothing, which a build says out loud.
var turns = map[string]turn{
	"mid-forward": {category: catalog.CategoryAmp, param: "Mid", steps: 1},
	"scooped":     {category: catalog.CategoryAmp, param: "Mid", steps: -1},

	"dark":   {category: catalog.CategoryAmp, param: "Treble", steps: -1},
	"bright": {category: catalog.CategoryAmp, param: "Treble", steps: 1},
	"glassy": {category: catalog.CategoryAmp, param: "Treble", steps: 1},

	"clean":          {category: catalog.CategoryAmp, param: "Drive", steps: -1},
	"minimal-drive":  {category: catalog.CategoryAmp, param: "Drive", steps: -0.5},
	"grit-on-attack": {category: catalog.CategoryAmp, param: "Drive", steps: 0.5},
	"saturated":      {category: catalog.CategoryAmp, param: "Drive", steps: 1},

	"tight-low-end": {category: catalog.CategoryAmp, param: "Sag", steps: -1},
	"loose-low-end": {category: catalog.CategoryAmp, param: "Sag", steps: 1},

	// How much of the room is on the part, which is the reverb's own
	// question and nothing to do with the amplifier.
	"dry": {
		category: catalog.CategoryReverb, param: "Mix", steps: -1,
		absenceMeans: "this chain has no reverb, so it is already dry",
	},
	"roomy": {category: catalog.CategoryReverb, param: "Mix", steps: 1},

	// A compressor's attack decides how much of the front of a note gets
	// past it. Slow, and the pick is a sound of its own; fast, and notes
	// arrive rather than start. A scale, like drive.
	"soft-attack":         {category: catalog.CategoryComp, param: "Attack", steps: -1},
	"audible-pick-attack": {category: catalog.CategoryComp, param: "Attack", steps: 0.5},
	"percussive":          {category: catalog.CategoryComp, param: "Attack", steps: 1},
}

// fallbackStep is how far a term moves a parameter the corpus cannot measure.
//
// A tenth of the stated range. Enough to hear, small enough that being wrong
// about it costs little, and used only where too few presets hold this model
// for a spread to mean anything.
const fallbackStep = 0.1

// move applies a rig's character to whichever blocks answer for it.
//
// Each axis names the kind of block it speaks to, because a word is about a
// part of the sound and not about a box: "roomy" is the reverb's question and
// "mid-forward" is the amplifier's. A chain holding neither still reports the
// words, because a term that moves nothing is still something the rig said and
// reporting only the ones that worked would read as if the rest had.
func move(
	blocks []catalog.Block,
	built chain.Chain,
	terms []string,
	stats *corpus.Stats,
) []Moved {
	out := make([]Moved, 0, len(terms))
	contested := contested(terms)

	for _, term := range terms {
		// Two words from one axis are two answers to one question. Neither
		// is applied, because applying both lands back where it started and
		// reads as though the rig said nothing.
		if axis, ok := axisOf(term); ok && contested[axis] {
			out = append(out, Moved{Term: term, Against: axis})

			continue
		}

		t, ok := turns[term]
		if !ok {
			out = append(out, Moved{Term: term})

			continue
		}

		at := indexOf(blocks, t.category)
		if at < 0 {
			// A word can ask for what the chain already is. Mix at zero and
			// no reverb at all are the same signal, so a rig asking to stay
			// dry got what it asked for and nothing is missing.
			if t.absenceMeans != "" {
				out = append(out, Moved{Term: term, Already: t.absenceMeans})

				continue
			}

			// Otherwise something would answer for this word and this chain
			// holds none of it, which is the rig's shape rather than a gap
			// here.
			out = append(out, Moved{
				Term:    term,
				Because: "this chain holds no " + string(t.category),
			})

			continue
		}

		out = append(out, apply(blocks[at], built.Blocks[at].Params, term, t, stats))
	}

	return out
}

// indexOf finds the first block of a kind, or reports that there is none.
//
// The first, because a chain may hold two reverbs and a word is one opinion:
// spreading it over both would be two opinions nobody expressed.
func indexOf(blocks []catalog.Block, want catalog.Category) int {
	for i, b := range blocks {
		if b.Category == want {
			return i
		}
	}

	return -1
}

// apply turns one knob, or reports why it could not.
func apply(
	b catalog.Block,
	params chain.Params,
	term string,
	t turn,
	stats *corpus.Stats,
) Moved {
	p, held := b.Params[t.param]
	if !held {
		// Not every amplifier models sag, and an opto compressor has no
		// attack at all — the circuit decides it.
		return Moved{Term: term, Because: "the " + b.Name + " has no " + t.param}
	}

	from, ok := params[t.param].Float()
	if !ok {
		return Moved{Term: term, Because: "the " + b.Name + " has no " + t.param}
	}

	to := clamp(from+t.steps*step(b, t.param, p, stats), p.Min, p.Max)

	params[t.param] = catalog.Float(to)

	return Moved{Term: term, Param: t.param, From: from, To: to}
}

// step is how far one term moves this parameter.
//
// The corpus spread where there is one: a parameter every player sets the same
// way is one nobody has an opinion about, and a term should barely move it. A
// parameter players disagree about is one where an opinion is worth having.
// Line 6's range would say the same thing about both.
func step(
	b catalog.Block,
	key string,
	p catalog.Param,
	stats *corpus.Stats,
) float64 {
	if stats != nil {
		if d, ok := stats.Param(b.ID, key); ok && d.Spread() > 0 {
			return d.Spread()
		}
	}

	return (p.Max - p.Min) * fallbackStep
}

// clamp keeps a value inside what the device accepts.
func clamp(v, lo, hi float64) float64 {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

// termsOf reads the words a rig describes itself with, in the order written.
func termsOf(spec rig.Spec) []string {
	if spec.Character == nil {
		return nil
	}

	out := make([]string, 0, len(*spec.Character))
	for _, c := range *spec.Character {
		out = append(out, c.Term)
	}

	return out
}

// contested finds the axes a rig spoke for more than once.
func contested(terms []string) map[string]bool {
	seen := map[string]int{}

	for _, term := range terms {
		if axis, ok := axisOf(term); ok {
			seen[axis]++
		}
	}

	out := map[string]bool{}
	for axis, n := range seen {
		if n > 1 {
			out[axis] = true
		}
	}

	return out
}
