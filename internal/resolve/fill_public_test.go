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

package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/resolve"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/corpus"
)

type FillPublicTestSuite struct {
	suite.Suite
	cat *catalog.Catalog
}

func (s *FillPublicTestSuite) SetupSuite() {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// grammar builds statistics saying how often each category appears for bass.
func (s *FillPublicTestSuite) grammar(
	cats map[catalog.Category]corpus.CategoryStats,
	models map[catalog.ModelID]corpus.ModelStats,
) *corpus.Stats {
	return &corpus.Stats{
		Models:  models,
		Grammar: map[string]corpus.Grammar{"bass": {Chains: 100, Categories: cats}},
	}
}

func (s *FillPublicTestSuite) TestANearUniversalBlockIsAdded() {
	spec, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 90, Before: 90},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
		))

	s.Require().NoError(err)
	s.Require().Len(added, 1)
	s.Require().Equal("Minotaur", added[0].Block.Name)
	s.Require().InDelta(0.9, added[0].Share, 1e-9)
	// Drive feeds the amp's input, so it belongs ahead of it.
	s.Require().Equal(catalog.CategoryDrive, blockAt(s, spec, 0))
}

func (s *FillPublicTestSuite) TestABlockThatBelongsAfterTheAmpGoesAfter() {
	spec, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryReverb: {Chains: 95, After: 95},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_ReverbTest": {Uses: 40}},
		))

	s.Require().NoError(err)
	s.Require().Len(added, 1)
	s.Require().Equal(
		catalog.CategoryReverb, blockAt(s, spec, len(spec.Blocks)-1))
}

func (s *FillPublicTestSuite) TestNothingIsAddedWithoutStrongAgreement() {
	// A compressor in most chains is a convention worth following. One in
	// six-in-ten is a choice, and making it silently would be this tool
	// having opinions it cannot justify.
	_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 60, Before: 60},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
		))

	s.Require().NoError(err)
	s.Require().Empty(added)
}

func (s *FillPublicTestSuite) TestWhatTheRecipeAlreadyNamedIsNotDuplicated() {
	_, added, err := resolve.Resolve(recipe("Ampeg SVT", "", "Klon"), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
		))

	s.Require().NoError(err)
	s.Require().Empty(added, "the recipe asked for a drive and got the one it named")
}

func (s *FillPublicTestSuite) TestNothingIsAddedWhenThereIsNothingToAdd() {
	tests := []struct {
		name  string
		stats *corpus.Stats
	}{
		{"no statistics at all", nil},
		{
			"no chains measured for this instrument",
			&corpus.Stats{Grammar: map[string]corpus.Grammar{}},
		},
		{
			"a category the catalog has no model for",
			s.grammar(
				map[catalog.Category]corpus.CategoryStats{
					catalog.Category("nonsense"): {Chains: 99, Before: 99},
				},
				map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 1}},
			),
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, tc.stats)

			s.Require().NoError(err)
			s.Require().Empty(added)
		})
	}
}

func (s *FillPublicTestSuite) TestTheMostUsedModelIsChosen() {
	_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			map[catalog.ModelID]corpus.ModelStats{
				"HD2_DistMinotaur": {Uses: 40},
				"HD2_StereoDrive":  {Uses: 900},
			},
		))

	s.Require().NoError(err)
	s.Require().Len(added, 1)
	s.Require().Equal(catalog.ModelID("HD2_StereoDrive"), added[0].Block.ID,
		"when nobody named a pedal, the one most people reach for is the "+
			"only defensible choice")
}

func (s *FillPublicTestSuite) TestCategoriesUsedEquallyOftenDoNotShuffle() {
	// Two conventions of equal weight must fill in the same order every run,
	// or the same recipe yields a different rig for no reason anybody chose.
	stats := s.grammar(
		map[catalog.Category]corpus.CategoryStats{
			catalog.CategoryDrive:  {Chains: 90, Before: 90},
			catalog.CategoryReverb: {Chains: 90, After: 90},
		},
		map[catalog.ModelID]corpus.ModelStats{
			"HD2_DistMinotaur": {Uses: 40},
			"HD2_ReverbTest":   {Uses: 40},
		})

	var first []string

	for range 3 {
		_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, stats)
		s.Require().NoError(err)
		s.Require().Len(added, 2)

		got := []string{string(added[0].Block.ID), string(added[1].Block.ID)}
		if first == nil {
			first = got
		}

		s.Require().Equal(first, got)
	}
}

func (s *FillPublicTestSuite) TestModelsUsedEquallyOftenBreakOnIdentifier() {
	stats := s.grammar(
		map[catalog.Category]corpus.CategoryStats{
			catalog.CategoryDrive: {Chains: 90, Before: 90},
		},
		map[catalog.ModelID]corpus.ModelStats{
			"HD2_StereoDrive":  {Uses: 40},
			"HD2_DistMinotaur": {Uses: 40},
		})

	for range 3 {
		_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, stats)
		s.Require().NoError(err)
		s.Require().Len(added, 1)
		s.Require().Equal(catalog.ModelID("HD2_DistMinotaur"), added[0].Block.ID)
	}
}

func (s *FillPublicTestSuite) TestAModelTheCatalogNeverHeardOfIsNotChosen() {
	// The corpus measures whatever presets held; a catalog for one device
	// will not carry all of it, and a chain cannot use what the device lacks.
	_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_GhostModel": {Uses: 900}},
		))

	s.Require().NoError(err)
	s.Require().Empty(added)
}

func (s *FillPublicTestSuite) TestABlockNeedingTheOwnersOwnIRIsNotChosen() {
	// It carries a slot index, not audio, so a generated chain reaching for
	// one would point at whatever happened to be loaded there.
	_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryCab: {Chains: 95, After: 95},
			},
			map[catalog.ModelID]corpus.ModelStats{
				"HD2_ImpulseResponse1024": {Uses: 900},
			},
		))

	s.Require().NoError(err)
	s.Require().Empty(added)
}

func (s *FillPublicTestSuite) TestGearForTheOtherInstrumentIsNotChosen() {
	// Line 6 tags amps and cabinets Guitar or Bass. This amp names no
	// cabinet, so the chain has a cabinet-shaped hole — and a guitar cabinet
	// must not fill it.
	_, added, err := resolve.Resolve(recipe("Cabless Bass Head", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryCab: {Chains: 99, After: 99},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_CabGuitarOnly": {Uses: 900}},
		))

	s.Require().NoError(err)
	s.Require().Empty(added)
}

func (s *FillPublicTestSuite) TestABlockWhoseCostWasGuessedIsNotChosen() {
	// Validation refuses a chain budgeted on an inferred cost. Choosing one
	// here would produce a chain rejected a moment later, blaming a block
	// nobody asked for.
	_, added, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			map[catalog.ModelID]corpus.ModelStats{"HD2_NoDefault": {Uses: 900}},
		))

	s.Require().NoError(err)
	s.Require().Empty(added)
}

func (s *FillPublicTestSuite) TestAnUntaggedBlockSuitsEitherInstrument() {
	// Only amps and cabinets carry an instrument tag; a pedal serves either.
	_, added, err := resolve.Resolve(recipe("Cabless Bass Head", ""), s.cat,
		s.grammar(
			map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryCab:   {Chains: 99, After: 99},
				catalog.CategoryDrive: {Chains: 80, Before: 80},
			},
			map[catalog.ModelID]corpus.ModelStats{
				"HD2_CabGuitarOnly": {Uses: 900},
				"HD2_DistMinotaur":  {Uses: 40},
			},
		))

	s.Require().NoError(err)
	s.Require().Len(added, 1, "the guitar cabinet is refused, the pedal is not")
	s.Require().Equal(catalog.ModelID("HD2_DistMinotaur"), added[0].Block.ID)
}

// blockAt returns the category of the block at a position.
func blockAt(s *FillPublicTestSuite, spec chain.Chain, i int) catalog.Category {
	b, ok := s.cat.Block(spec.Blocks[i].Model)
	s.Require().True(ok)

	return b.Category
}

func TestFillPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FillPublicTestSuite))
}
