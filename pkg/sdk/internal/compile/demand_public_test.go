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
package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

type DemandPublicTestSuite struct {
	suite.Suite
	cat *catalog.Catalog
}

func (s *DemandPublicTestSuite) SetupSuite() {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// asking builds a bass rig that says something, without saying any gear
// beyond the amplifier.
//
// The amplifier is the SVT's normal channel, which carries a MidFreq and no
// Mid. That is the shape this whole file is about: an amp that answers most
// questions and not this one.
func asking(
	terms []string,
	attack rig.Attack,
	pedals ...string,
) rig.Spec {
	spec := recipe("Ampeg SVT", "", pedals...)

	if len(terms) > 0 {
		character := make([]rig.CharacterTerm, 0, len(terms))
		for _, t := range terms {
			character = append(character, rig.CharacterTerm{Term: t})
		}

		spec.Character = &character
	}

	if attack != "" {
		spec.Technique = &rig.Technique{Attack: attack}
	}

	return spec
}

// statistics say nothing is near-universal, so fill adds nothing and whatever
// arrives was demanded rather than filled.
//
// Counts are given for every model a case can reach, because commonest skips
// a model no chain for this instrument held.
func (s *DemandPublicTestSuite) quiet(
	cats map[catalog.Category]corpus.CategoryStats,
	models ...catalog.ModelID,
) *corpus.Stats {
	byModel := map[catalog.ModelID]corpus.ModelStats{}
	counts := map[catalog.ModelID]int{}

	for i, m := range models {
		// Descending, so a case offering two models of a kind has a winner
		// that is not a tie broken on the identifier.
		byModel[m] = corpus.ModelStats{Uses: 100 - i}
		counts[m] = 100 - i
	}

	return &corpus.Stats{
		Models: byModel,
		Grammar: map[string]corpus.Grammar{
			"bass": {Chains: 100, Categories: cats, Models: counts},
		},
	}
}

// TestResolveDemand covers blocks that are in a chain because the rig asked
// for them.
//
// The distinction from filling is the whole point. Filling answers what a
// chain of this kind usually has; this answers what this rig said it needs,
// and before it existed a word could only reach a control that happened to
// be there already.
func (s *DemandPublicTestSuite) TestResolveDemand() {
	tests := []struct {
		name string

		terms  []string
		attack rig.Attack
		pedals []string
		models []catalog.ModelID
		// where the corpus puts each kind, for the cases that check which
		// side of the amplifier a demanded block lands on.
		cats map[catalog.Category]corpus.CategoryStats
		// no statistics at all.
		none bool
		// statistics measuring no chains for this instrument.
		silent bool

		wantIDs []catalog.ModelID
		// a fragment the reason for the first demanded block must carry.
		wantReason string
		// the category of the first block in the chain, and of the last.
		first catalog.Category
		last  catalog.Category
	}{
		{
			// The case this was written for. The SVT's normal channel has a
			// MidFreq and no Mid, so the word that earned its place by
			// measurement had nowhere to land.
			name:       "a word whose control no block in the chain carries",
			terms:      []string{"mid-forward"},
			models:     []catalog.ModelID{"HD2_EQTestParametric"},
			wantIDs:    []catalog.ModelID{"HD2_EQTestParametric"},
			wantReason: "the rig says mid-forward and nothing here had a MidGain",
		},
		{
			// Both sides of the axis ask the same question of the same band.
			name:    "the other word on that axis",
			terms:   []string{"scooped"},
			models:  []catalog.ModelID{"HD2_EQTestParametric"},
			wantIDs: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// An equaliser is not a Mid just because it is an equaliser.
			// Seating one without the band would answer nothing and cost
			// DSP.
			name:   "an equaliser that has no such band",
			terms:  []string{"mid-forward"},
			models: []catalog.ModelID{"HD2_EQTestNoMid"},
		},
		{
			// The device has nothing that carries the band. Reported by the
			// word rather than papered over with a block that cannot help.
			name:   "a band this device has nowhere to put",
			terms:  []string{"mid-forward"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
		},
		{
			// A word naming a block asserts the block is there. It is not a
			// setting, so it asks for a filter and turns nothing up.
			name:       "a word that names a block rather than a setting",
			terms:      []string{"envelope-swept"},
			models:     []catalog.ModelID{"HD2_FilterTestMutant"},
			wantIDs:    []catalog.ModelID{"HD2_FilterTestMutant"},
			wantReason: "the rig says envelope-swept, which needs one",
		},
		{
			// How the instrument is played is the same kind of claim as a
			// word that names a block, and arrives from a different field.
			name:       "slap, which is not the sound without compression",
			attack:     rig.AttackSlap,
			models:     []catalog.ModelID{"HD2_CompTestDeluxe"},
			wantIDs:    []catalog.ModelID{"HD2_CompTestDeluxe"},
			wantReason: "the rig says slap, which needs one",
		},
		{
			// Every rig names an attack and most name nothing this reads.
			name:   "an attack that asks for nothing",
			attack: rig.AttackPick,
			models: []catalog.ModelID{"HD2_CompTestDeluxe"},
		},
		{
			// Asking for what is already there would seat a second one.
			name:   "a filter the rig already named",
			terms:  []string{"envelope-swept"},
			pedals: []string{"Mu-Tron III"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
		},
		{
			// Two claims wanting the same kind of block get one block. The
			// word is read after the technique has seated it.
			name:    "two claims asking for the same kind of block",
			terms:   []string{"percussive"},
			attack:  rig.AttackSlap,
			models:  []catalog.ModelID{"HD2_CompTestDeluxe"},
			wantIDs: []catalog.ModelID{"HD2_CompTestDeluxe"},
		},
		{
			// A term nothing acts on asks for nothing. These are the axes
			// that describe the player rather than the signal.
			name:   "a term that moves nothing and names nothing",
			terms:  []string{"bridge-forward"},
			models: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// Mix at zero and no reverb are the same signal, so the chain
			// already answers this and nothing is added.
			name:   "a word the chain's own shape answers",
			terms:  []string{"dry"},
			models: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{
			// A demanded block sits where the corpus puts that kind, the
			// same question fill asks of the blocks it adds. A filter mostly
			// seen ahead of the amplifier goes ahead of it.
			name:   "a demanded block the corpus puts before the amp",
			terms:  []string{"envelope-swept"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryFilter: {Chains: 40, Before: 40},
			},
			wantIDs: []catalog.ModelID{"HD2_FilterTestMutant"},
			first:   catalog.CategoryFilter,
		},
		{
			name:   "a demanded block the corpus puts after the amp",
			terms:  []string{"envelope-swept"},
			models: []catalog.ModelID{"HD2_FilterTestMutant"},
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryFilter: {Chains: 40, After: 40},
			},
			wantIDs: []catalog.ModelID{"HD2_FilterTestMutant"},
			last:    catalog.CategoryFilter,
		},
		{
			// Two words from one axis cancel, and a cancelled word must not
			// drag a block in on its way out.
			name:   "both words from one axis",
			terms:  []string{"mid-forward", "scooped"},
			models: []catalog.ModelID{"HD2_EQTestParametric"},
		},
		{name: "no statistics at all", terms: []string{"mid-forward"}, none: true},
		{
			name:   "no chains measured for this instrument",
			terms:  []string{"mid-forward"},
			silent: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var stats *corpus.Stats

			switch {
			case tt.none:
			case tt.silent:
				stats = &corpus.Stats{Grammar: map[string]corpus.Grammar{}}
			default:
				stats = s.quiet(tt.cats, tt.models...)
			}

			built, added, _, err := compile.Resolve(
				asking(tt.terms, tt.attack, tt.pedals...), s.cat, stats)

			s.Require().NoError(err)

			got := make([]catalog.ModelID, 0, len(added))
			for _, a := range added {
				got = append(got, a.Block.ID)
			}

			s.Require().Len(got, len(tt.wantIDs))

			if tt.wantIDs != nil {
				s.Require().Equal(tt.wantIDs, got)
			}

			if tt.wantReason != "" {
				s.Require().Equal(tt.wantReason, added[0].Reason)
			}

			if tt.first != "" {
				s.Require().Equal(tt.first, s.categoryAt(built, 0))
			}

			if tt.last != "" {
				s.Require().Equal(
					tt.last, s.categoryAt(built, len(built.Blocks)-1))
			}
		})
	}
}

// TestResolveDemandIsDeterministic guards the order blocks arrive in.
//
// Two claims of equal weight must seat the same block every run, or the same
// rig yields a different preset for no reason anybody chose.
func (s *DemandPublicTestSuite) TestResolveDemandIsDeterministic() {
	spec := asking([]string{"mid-forward", "envelope-swept"}, rig.AttackSlap)
	stats := s.quiet(
		nil, "HD2_EQTestParametric", "HD2_FilterTestMutant", "HD2_CompTestDeluxe")

	var first []catalog.ModelID

	for range 5 {
		_, added, _, err := compile.Resolve(spec, s.cat, stats)
		s.Require().NoError(err)

		got := make([]catalog.ModelID, 0, len(added))
		for _, a := range added {
			got = append(got, a.Block.ID)
		}

		if first == nil {
			first = got
		}

		s.Require().Equal(first, got, "the same rig must demand the same way")
	}

	s.Require().Len(first, 3)
}

// TestResolveDemandLandsTheWord is the end the rest of this exists for.
//
// Seating an equaliser is not the point. The point is that a word which
// measured a record now reaches a control, so the check is on what moved
// rather than on what was added.
func (s *DemandPublicTestSuite) TestResolveDemandLandsTheWord() {
	_, _, moved, err := compile.Resolve(
		asking([]string{"mid-forward"}, ""),
		s.cat,
		s.quiet(nil, "HD2_EQTestParametric"),
	)
	s.Require().NoError(err)
	s.Require().Len(moved, 1)

	s.Require().Equal("mid-forward", moved[0].Term)
	s.Require().Equal("MidGain", moved[0].Param)
	s.Require().Greater(moved[0].To, moved[0].From)
	s.Require().Empty(moved[0].Because, "the word landed, so nothing excuses it")
}

// TestResolveDemandNamesTheBlock covers what a word naming a block reports.
//
// "satisfied" invites nobody to check. Which filter got seated is the part
// somebody reading the preset can disagree with, so it is what gets said.
func (s *DemandPublicTestSuite) TestResolveDemandNamesTheBlock() {
	tests := []struct {
		name    string
		models  []catalog.ModelID
		already string
		because string
	}{
		{
			name:    "the block is there",
			models:  []catalog.ModelID{"HD2_FilterTestMutant"},
			already: "the Test Mutant Filter is what this asks for",
		},
		{
			name:    "this device has nothing of the kind",
			models:  []catalog.ModelID{"HD2_EQTestParametric"},
			because: "this device has no filter",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, _, moved, err := compile.Resolve(
				asking([]string{"envelope-swept"}, ""),
				s.cat,
				s.quiet(nil, tt.models...),
			)
			s.Require().NoError(err)
			s.Require().Len(moved, 1)

			s.Require().Equal(tt.already, moved[0].Already)
			s.Require().Equal(tt.because, moved[0].Because)
		})
	}
}

// categoryAt returns the category of the block at a position.
func (s *DemandPublicTestSuite) categoryAt(
	built chain.Chain,
	i int,
) catalog.Category {
	b, ok := s.cat.Block(built.Blocks[i].Model)
	s.Require().True(ok)

	return b.Category
}

func TestDemandPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DemandPublicTestSuite))
}
