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

// amp is a block carrying the four controls a term can turn.
func (s *MoveTestSuite) amp() catalog.Block {
	knob := catalog.Param{Type: catalog.ParamFloat, Min: 0, Max: 1}

	return catalog.Block{
		ID: "HD2_AmpTestBass", Category: catalog.CategoryAmp,
		Params: map[string]catalog.Param{
			"Mid": knob, "Treble": knob, "Drive": knob, "Sag": knob,
		},
	}
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
			name:  "a word the vocabulary does not carry at all",
			terms: []string{"sounds like a wet paper bag"},
			inert: true,
		},
		{
			name:  "an axis nothing acts on yet",
			terms: []string{"short-decay"},
			inert: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			params := s.params()

			got := move(s.amp(), params, tt.terms, s.stats())

			s.Require().Len(got, len(tt.terms))

			if tt.inert {
				s.Require().False(got[0].Acted())
				s.Require().False(got[0].Contested())

				return
			}

			if tt.contested != "" {
				for _, m := range got {
					s.Require().True(m.Contested())
					s.Require().Equal(tt.contested, m.Against)
					s.Require().False(m.Acted())
				}
			}

			v, ok := params[tt.param].Float()
			s.Require().True(ok)
			s.Require().InDelta(tt.want, v, 1e-9)
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
			params := chain.Params{"Drive": catalog.Float(tt.at)}

			got := move(s.amp(), params, []string{tt.term}, s.stats())

			s.Require().True(got[0].Acted())

			v, _ := params["Drive"].Float()
			s.Require().InDelta(tt.want, v, 1e-9)
		})
	}
}

// TestMoveSkipsWhatTheBlockDoesNotHave covers an amplifier that models no sag.
func (s *MoveTestSuite) TestMoveSkipsWhatTheBlockDoesNotHave() {
	b := catalog.Block{
		ID: "HD2_Plain", Category: catalog.CategoryAmp,
		Params: map[string]catalog.Param{
			"Mid": {Type: catalog.ParamFloat, Min: 0, Max: 1},
		},
	}

	got := move(b, chain.Params{"Mid": catalog.Float(0.5)},
		[]string{"tight-low-end"}, nil)

	s.Require().Len(got, 1)
	s.Require().False(got[0].Acted())
}

// TestMoveWithoutStatistics covers a build with no corpus to lean on.
func (s *MoveTestSuite) TestMoveWithoutStatistics() {
	params := s.params()

	got := move(s.amp(), params, []string{"mid-forward"}, nil)

	s.Require().True(got[0].Acted())

	// A tenth of the range, since nothing measured this.
	v, _ := params["Mid"].Float()
	s.Require().InDelta(0.6, v, 1e-9)
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

	got := move(b, chain.Params{"Mid": catalog.Bool(true)},
		[]string{"mid-forward"}, nil)

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
