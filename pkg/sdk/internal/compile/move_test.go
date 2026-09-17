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
func (s *MoveTestSuite) paramOf(
	built chain.Chain,
	key string,
) float64 {
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

// TestStep covers how far one word moves a knob.
func (s *MoveTestSuite) TestStep() {
	wide := catalog.Param{Type: catalog.ParamFloat, Min: 0, Max: 10}

	tests := []struct {
		name  string
		p     catalog.Param
		stats *corpus.Stats
		want  float64
	}{
		{
			// Players mostly agree, so the spread is the step.
			name:  "a spread narrower than the cap",
			p:     knob,
			stats: s.spread(0.46, 0.54),
			want:  0.08,
		},
		{
			// Sag on the Cali 400 spreads across half its range. One word
			// would put it on the rail, so the step stops at a quarter.
			name:  "a spread wider than the cap",
			p:     knob,
			stats: s.spread(0.25, 0.75),
			want:  0.25,
		},
		{
			name:  "a cap measured against the control's own range",
			p:     wide,
			stats: s.spread(1, 9),
			want:  2.5,
		},
		{
			name:  "players who all set it the same way",
			p:     knob,
			stats: s.spread(0.5, 0.5),
			want:  0.1,
		},
		{name: "no statistics at all", p: knob, want: 0.1},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := step(s.amp(), "Sag", tt.p, tt.stats)

			s.Require().InDelta(tt.want, got, 1e-9)
		})
	}
}

// spread says players set the test amplifier's Sag between p25 and p75.
func (s *MoveTestSuite) spread(
	p25 float64,
	p75 float64,
) *corpus.Stats {
	return &corpus.Stats{
		Models: map[catalog.ModelID]corpus.ModelStats{
			"HD2_AmpTestBass": {
				Uses: 40,
				Params: map[string]corpus.ParamStats{
					"Sag": {N: 40, Median: (p25 + p75) / 2, P25: p25, P75: p75},
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

			got := move(s.blocks(), built, said(tt.terms...), s.stats())

			s.Require().Len(got, len(tt.terms))

			if tt.inert {
				s.Require().False(got[0].Acted())
				s.Require().False(got[0].Contested())
				// This chain answers every axis that acts, so silence here
				// is the project's and not the rig's.
				s.Require().False(got[0].Unanswered())
				s.Require().False(got[0].Holds())

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

			got := move([]catalog.Block{s.amp()}, built, said(tt.term), s.stats())

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

	got := move([]catalog.Block{b}, built, said("tight-low-end"), nil)

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
	s.Require().True(got[0].Unanswered())
	s.Require().Equal("the Plain Amp has no Sag", got[0].Because)
}

// TestMoveWhenTheChainIsAlreadyWhatTheWordAsked covers asking for what you
// already have.
//
// Mix at zero and no reverb at all are the same signal. A rig asking to stay
// dry, in a chain holding no reverb, got what it asked for, and saying the
// chain could not answer would be backwards.
func (s *MoveTestSuite) TestMoveWhenTheChainIsAlreadyWhatTheWordAsked() {
	built := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTestBass", Params: s.params()},
	}}

	got := move([]catalog.Block{s.amp()}, built, said("dry"), s.stats())

	s.Require().Len(got, 1)
	s.Require().True(got[0].Holds())
	s.Require().False(got[0].Acted())
	s.Require().False(got[0].Unanswered())
	s.Require().Equal("this chain has no reverb, so it is already dry", got[0].Already)
}

// TestMoveTurnsTheReverbThatIsThere covers the same word with somewhere to go.
//
// Absence answers dry; a reverb in the chain does not, and the word has to
// reach for the knob.
func (s *MoveTestSuite) TestMoveTurnsTheReverbThatIsThere() {
	built := s.built()

	got := move(s.blocks(), built, said("dry"), s.stats())

	s.Require().Len(got, 1)
	s.Require().False(got[0].Holds())
	s.Require().True(got[0].Acted())
	s.Require().Less(s.paramOf(built, "Mix"), 0.5)
}

// TestMoveWithoutTheBlockTheWordNeeds covers a word with nowhere to land.
//
// A rig asking for room in a chain holding no reverb is not a word nobody has
// taught the project: it is a chain that cannot answer.
func (s *MoveTestSuite) TestMoveWithoutTheBlockTheWordNeeds() {
	built := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTestBass", Params: s.params()},
	}}

	got := move([]catalog.Block{s.amp()}, built, said("roomy"), s.stats())

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
	s.Require().True(got[0].Unanswered())
	s.Require().Equal("this chain holds no reverb", got[0].Because)
}

// TestMoveWithoutStatistics covers a build with no corpus to lean on.
func (s *MoveTestSuite) TestMoveWithoutStatistics() {
	built := s.built()

	got := move(s.blocks(), built, said("mid-forward"), nil)

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

	got := move([]catalog.Block{b}, built, said("mid-forward"), nil)

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
}

// TestTermsOf covers reading the words a rig described itself with, and how
// much of a step each one is worth.
func (s *MoveTestSuite) TestTermsOf() {
	figures := func(mine, theirs float64, key string) *[]rig.Evidence {
		return &[]rig.Evidence{{
			Kind:     rig.EvidenceAudio,
			Measured: &map[string]float64{key: mine},
			Against:  &map[string]float64{key: theirs},
		}}
	}

	tests := []struct {
		name string
		in   *[]rig.CharacterTerm
		want []heard
	}{
		{name: "a rig that said nothing about how it sounds"},
		{
			// Nobody measured these, so each is worth the whole step, which
			// is what every rig did before any of this could be weighed.
			name: "the words, in the order they were written",
			in: &[]rig.CharacterTerm{
				{Term: "mid-forward"}, {Term: "saturated"},
			},
			want: []heard{
				{term: "mid-forward", weight: 1}, {term: "saturated", weight: 1},
			},
		},
		{
			// Half the harmonics of everybody else is half a step. The word
			// says which way; the gap says how far.
			name: "a word with the gap that earned it",
			in: &[]rig.CharacterTerm{
				{Term: "clean", Evidence: figures(0.12, 0.24, "harmonics")},
			},
			want: []heard{{term: "clean", weight: 0.5}},
		},
		{
			// A gap wider than what the others read is still one word, and
			// one word does not decide the whole of a control.
			name: "a gap wider than the others' own figure",
			in: &[]rig.CharacterTerm{
				{Term: "mid-forward", Evidence: figures(0.09, 0.02, "mid")},
			},
			want: []heard{{term: "mid-forward", weight: 1}},
		},
		{
			name: "a centroid, which the highs are earned from",
			in: &[]rig.CharacterTerm{
				{Term: "dark", Evidence: figures(96, 170, "centroid")},
			},
			want: []heard{{term: "dark", weight: 0.43529411764705883}},
		},
		{
			// Every way the figures can fail to say anything, each of which
			// leaves the word worth its whole step rather than nothing.
			name: "figures that do not weigh the word",
			in: &[]rig.CharacterTerm{
				// An axis no measurement speaks to.
				{Term: "quiet-strings", Evidence: figures(0.1, 0.2, "harmonics")},
				// A word the vocabulary does not carry.
				{Term: "chewy", Evidence: figures(0.1, 0.2, "harmonics")},
				// One side of the comparison missing.
				{Term: "clean", Evidence: &[]rig.Evidence{{
					Kind: rig.EvidenceAudio, Measured: &map[string]float64{"harmonics": 0.12},
				}}},
				// The wrong measure for this axis.
				{Term: "bright", Evidence: figures(0.12, 0.24, "harmonics")},
				// Nothing to compare against but zero.
				{Term: "scooped", Evidence: figures(0.0, 0.0, "mid")},
				// No evidence at all.
				{Term: "saturated"},
			},
			want: []heard{
				{term: "quiet-strings", weight: 1},
				{term: "chewy", weight: 1},
				{term: "clean", weight: 1},
				{term: "bright", weight: 1},
				{term: "scooped", weight: 1},
				{term: "saturated", weight: 1},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := termsOf(rig.Spec{Character: tt.in})

			s.Require().Equal(tt.want, got)
		})
	}
}

// TestAMeasuredWordMovesLessThanAnAssertedOne is the point of weighing a
// term: the same word, on the same chain, moves further when the gap that
// earned it is wider.
func (s *MoveTestSuite) TestAMeasuredWordMovesLessThanAnAssertedOne() {
	asserted := s.built()
	measured := s.built()

	s.Require().NotEmpty(move(s.blocks(), asserted, said("clean"), nil))
	s.Require().NotEmpty(move(s.blocks(), measured,
		[]heard{{term: "clean", weight: 0.5}}, nil))

	was, _ := s.built().Blocks[1].Params["Drive"].Float()
	full, _ := asserted.Blocks[1].Params["Drive"].Float()
	half, _ := measured.Blocks[1].Params["Drive"].Float()

	s.Require().Less(full, half, "half a step lands nearer where it started")
	s.Require().Less(half, was, "and still moves the control the way the word says")
	s.Require().InDelta(was-half, (was-full)/2, 1e-9,
		"half the weight is half the distance")
}

func TestMoveTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MoveTestSuite))
}

// said is a set of terms nobody measured, which is what every rig carried
// before a measurement could size a move.
func said(
	terms ...string,
) []heard {
	out := make([]heard, 0, len(terms))
	for _, term := range terms {
		out = append(out, heard{term: term, weight: 1})
	}

	return out
}
