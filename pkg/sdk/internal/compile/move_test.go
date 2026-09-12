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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

type MoveTestSuite struct {
	suite.Suite
}

// knob is a control that runs from nothing to everything.
var knob = catalog.Param{Type: catalog.ParamFloat, Min: 0, Max: 1}

// amp carries the four controls an amplifier term can turn.
func (s *MoveTestSuite) amp() catalog.Block {
	return catalog.Block{
		ID: "HD2_AmpTestBass", Category: catalog.CategoryAmp,
		Params: map[string]catalog.Param{
			"Mid": knob, "Treble": knob, "Drive": knob, "Sag": knob,
		},
	}
}

// chain is an amplifier, a reverb and a compressor, which between them answer
// for every axis that acts.
func (s *MoveTestSuite) blocks() []catalog.Block {
	return []catalog.Block{
		{
			ID: "HD2_CompTest", Category: catalog.CategoryComp,
			Params: map[string]catalog.Param{"Attack": knob},
		},
		s.amp(),
		{
			ID: "HD2_ReverbTest", Category: catalog.CategoryReverb,
			Params: map[string]catalog.Param{"Mix": knob},
		},
	}
}

// built is where the corpus and the catalog left every block, before a word.
func (s *MoveTestSuite) built() chain.Chain {
	return chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_CompTest", Params: chain.Params{"Attack": catalog.Float(0.5)}},
		{Model: "HD2_AmpTestBass", Params: s.params()},
		{Model: "HD2_ReverbTest", Params: chain.Params{"Mix": catalog.Float(0.5)}},
	}}
}

// paramOf reads one control off whichever block holds it.
func (s *MoveTestSuite) paramOf(built chain.Chain, key string) float64 {
	for _, b := range built.Blocks {
		if v, ok := b.Params[key].Float(); ok {
			return v
		}
	}

	s.Require().Fail("nothing in the chain has " + key)

	return 0
}

// params is where the corpus and the catalog left things, before any word.
func (s *MoveTestSuite) params() chain.Params {
	return chain.Params{
		"Mid":    catalog.Float(0.5),
		"Treble": catalog.Float(0.5),
		"Drive":  catalog.Float(0.5),
		"Sag":    catalog.Float(0.5),
	}
}

// stats say players disagree about Mid by 0.08 and about nothing else.
func (s *MoveTestSuite) stats() *corpus.Stats {
	return &corpus.Stats{
		Models: map[catalog.ModelID]corpus.ModelStats{
			"HD2_AmpTestBass": {
				Uses: 40,
				Params: map[string]corpus.ParamStats{
					"Mid": {N: 40, Median: 0.5, P25: 0.46, P75: 0.54},
				},
			},
		},
	}
}

// TestMove covers what a word does to a knob.
func (s *MoveTestSuite) TestMove() {
	tests := []struct {
		name  string
		terms []string
		// the parameter that must have moved, and where to.
		param string
		want  float64
		// the term must be recorded as having moved nothing.
		inert     bool
		contested string
	}{
		{
			// The corpus says players disagree about Mid by 0.08, so one
			// step is 0.08.
			name:  "a pair term moves by what players disagree about",
			terms: []string{"mid-forward"},
			param: "Mid", want: 0.58,
		},
		{
			name:  "and the other way for the other half of the pair",
			terms: []string{"scooped"},
			param: "Mid", want: 0.42,
		},
		{
			// Nobody measured Treble, so the step is a tenth of the range.
			name:  "a parameter the corpus cannot measure falls back to the range",
			terms: []string{"bright"},
			param: "Treble", want: 0.6,
		},
		{
			// clean, minimal-drive, grit-on-attack, saturated are four
			// points on one line, so each carries its own multiple.
			name:  "a scale term moves its own share of a step",
			terms: []string{"minimal-drive"},
			param: "Drive", want: 0.45,
		},
		{
			name:  "and the far end of the same scale moves a whole one",
			terms: []string{"saturated"},
			param: "Drive", want: 0.6,
		},
		{
			// The Pilot's Guide: lower values offer tighter responsiveness.
			name:  "sag, which the guide had to explain",
			terms: []string{"tight-low-end"},
			param: "Sag", want: 0.4,
		},
		{
			// Two answers to one question. Applying both would land back
			// where it started and read as though the rig said nothing.
			name:      "two words from one axis move nothing",
			terms:     []string{"minimal-drive", "grit-on-attack"},
			param:     "Drive",
			want:      0.5,
			contested: "drive",
		},
		{
			name:  "a word on an axis no amplifier control answers to",
			terms: []string{"glassy"},
			param: "Treble", want: 0.6,
		},
		{
			// The reverb's question, not the amplifier's.
			name:  "room around the part",
			terms: []string{"roomy"},
			param: "Mix", want: 0.6,
		},
		{
			name:  "none on it",
			terms: []string{"dry"},
			param: "Mix", want: 0.4,
		},
		{
			// A compressor's attack decides how much of the front of a note
			// gets past it. Slow lets the pick through; fast clamps it.
			name:  "the pick as a sound of its own",
			terms: []string{"percussive"},
			param: "Attack", want: 0.6,
		},
		{
			name:  "notes that arrive rather than start",
			terms: []string{"soft-attack"},
			param: "Attack", want: 0.4,
		},
		{
			name:  "a word the vocabulary does not carry at all",
			terms: []string{"sounds like a wet paper bag"},
			inert: true,
		},
		{
			// Its own words say the hands and the strings do this, and the
			// rig has three plausible controls for it and no way to choose.
			name:  "an axis nothing acts on yet",
			terms: []string{"short-decay"},
			inert: true,
		},
		{
			// What the hands make is not something a knob answers for.
			name:  "an axis about the player rather than the rig",
			terms: []string{"audible-strings"},
			inert: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			built := s.built()

			got := move(s.blocks(), built, tt.terms, s.stats())

			s.Require().Len(got, len(tt.terms))

			if tt.inert {
				s.Require().False(got[0].Acted())
				s.Require().False(got[0].Contested())
				// This chain answers every axis that acts, so silence here
				// is the project's and not the rig's.
				s.Require().False(got[0].Unanswered())

				return
			}

			if tt.contested != "" {
				for _, m := range got {
					s.Require().True(m.Contested())
					s.Require().Equal(tt.contested, m.Against)
					s.Require().False(m.Acted())
				}
			}

			s.Require().InDelta(tt.want, s.paramOf(built, tt.param), 1e-9)
		})
	}
}

// TestMoveClampsToWhatTheDeviceAccepts covers a word asking for more than
// there is.
//
// A term is an opinion about direction, not a promise that the range is deep
// enough to hold it.
func (s *MoveTestSuite) TestMoveClampsToWhatTheDeviceAccepts() {
	tests := []struct {
		name string
		at   float64
		term string
		want float64
	}{
		{name: "already at the top", at: 1, term: "saturated", want: 1},
		{name: "already at the bottom", at: 0, term: "clean", want: 0},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			built := chain.Chain{Blocks: []chain.Block{
				{Model: "HD2_AmpTestBass", Params: chain.Params{
					"Drive": catalog.Float(tt.at),
				}},
			}}

			got := move([]catalog.Block{s.amp()}, built, []string{tt.term}, s.stats())

			s.Require().True(got[0].Acted())
			s.Require().InDelta(tt.want, s.paramOf(built, "Drive"), 1e-9)
		})
	}
}

// TestMoveSkipsWhatTheBlockDoesNotHave covers an amplifier that models no sag.
//
// The word is answerable and this amplifier cannot answer it, which is worth
// saying out loud rather than passing over.
func (s *MoveTestSuite) TestMoveSkipsWhatTheBlockDoesNotHave() {
	b := catalog.Block{
		ID: "HD2_Plain", Name: "Plain Amp", Category: catalog.CategoryAmp,
		Params: map[string]catalog.Param{
			"Mid": {Type: catalog.ParamFloat, Min: 0, Max: 1},
		},
	}

	built := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_Plain", Params: chain.Params{"Mid": catalog.Float(0.5)}},
	}}

	got := move([]catalog.Block{b}, built, []string{"tight-low-end"}, nil)

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
	s.Require().True(got[0].Unanswered())
	s.Require().Equal("the Plain Amp has no Sag", got[0].Because)
}

// TestMoveWithoutTheBlockTheWordNeeds covers a word with nowhere to land.
//
// A rig asking for room in a chain holding no reverb is not a word nobody has
// taught the project: it is a chain that cannot answer.
func (s *MoveTestSuite) TestMoveWithoutTheBlockTheWordNeeds() {
	built := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTestBass", Params: s.params()},
	}}

	got := move([]catalog.Block{s.amp()}, built, []string{"roomy"}, s.stats())

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
	s.Require().True(got[0].Unanswered())
	s.Require().Equal("this chain holds no reverb", got[0].Because)
}

// TestMoveWithoutStatistics covers a build with no corpus to lean on.
func (s *MoveTestSuite) TestMoveWithoutStatistics() {
	built := s.built()

	got := move(s.blocks(), built, []string{"mid-forward"}, nil)

	s.Require().True(got[0].Acted())

	// A tenth of the range, since nothing measured this.
	s.Require().InDelta(0.6, s.paramOf(built, "Mid"), 1e-9)
}

// TestMoveSkipsAValueItCannotDo covers a control a word cannot turn.
//
// A switch has no middle, so a term asking for more of it is asking for
// something the device would refuse.
func (s *MoveTestSuite) TestMoveSkipsAValueItCannotDo() {
	b := catalog.Block{
		ID: "HD2_Switched", Category: catalog.CategoryAmp,
		Params: map[string]catalog.Param{
			"Mid": {Type: catalog.ParamBool},
		},
	}

	built := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_Switched", Params: chain.Params{"Mid": catalog.Bool(true)}},
	}}

	got := move([]catalog.Block{b}, built, []string{"mid-forward"}, nil)

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
}

// TestTermsOf covers reading the words a rig described itself with.
func (s *MoveTestSuite) TestTermsOf() {
	tests := []struct {
		name string
		in   *[]rig.CharacterTerm
		want []string
	}{
		{name: "a rig that said nothing about how it sounds"},
		{
			name: "the words, in the order they were written",
			in: &[]rig.CharacterTerm{
				{Term: "mid-forward"}, {Term: "saturated"},
			},
			want: []string{"mid-forward", "saturated"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := termsOf(rig.Spec{Character: tt.in})

			s.Require().Equal(tt.want, got)
		})
	}
}

func TestMoveTestSuite(t *testing.T) {
	suite.Run(t, new(MoveTestSuite))
}
