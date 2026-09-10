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

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/compile"
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

// TestResolveFill covers what the corpus adds to a chain nobody asked for.
//
// Every case is resolved three times: two conventions of equal weight must
// fill in the same order every run, or the same recipe yields a different rig
// for no reason anybody chose.
func (s *FillPublicTestSuite) TestResolveFill() {
	tests := []struct {
		name string
		// the amp the recipe names, and anything else it names beside it.
		gear  string
		extra []string

		cats   map[catalog.Category]corpus.CategoryStats
		models map[catalog.ModelID]corpus.ModelStats
		// no statistics at all.
		none bool
		// statistics measuring no chains for this instrument.
		silent bool

		want      int
		wantIDs   []catalog.ModelID
		wantName  string
		wantShare float64
		// the category of the first block in the chain, and of the last.
		first catalog.Category
		last  catalog.Category
	}{
		{
			name: "a block nearly every chain has",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 90, Before: 90},
			},
			models:    map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
			want:      1,
			wantName:  "Minotaur",
			wantShare: 0.9,
			// Drive feeds the amp's input, so it belongs ahead of it.
			first: catalog.CategoryDrive,
		},
		{
			name: "a block that belongs after the amp",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryReverb: {Chains: 95, After: 95},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_ReverbTest": {Uses: 40}},
			want:   1,
			last:   catalog.CategoryReverb,
		},
		{
			// A compressor in most chains is a convention worth following.
			// One in six-in-ten is a choice, and making it silently would be
			// this tool having opinions it cannot justify.
			name: "agreement too weak to act on",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 60, Before: 60},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
		},
		{
			name:  "a convention the recipe already named a pedal for",
			extra: []string{"Klon"},
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 40}},
		},
		{name: "no statistics at all", none: true},
		{name: "no chains measured for this instrument", silent: true},
		{
			name: "a category the catalog has no model for",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.Category("nonsense"): {Chains: 99, Before: 99},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_DistMinotaur": {Uses: 1}},
		},
		{
			// When nobody named a pedal, the one most people reach for is the
			// only defensible choice.
			name: "two models for the convention, one of them common",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			models: map[catalog.ModelID]corpus.ModelStats{
				"HD2_DistMinotaur": {Uses: 40},
				"HD2_StereoDrive":  {Uses: 900},
			},
			want:    1,
			wantIDs: []catalog.ModelID{"HD2_StereoDrive"},
		},
		{
			name: "two categories used equally often",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive:  {Chains: 90, Before: 90},
				catalog.CategoryReverb: {Chains: 90, After: 90},
			},
			models: map[catalog.ModelID]corpus.ModelStats{
				"HD2_DistMinotaur": {Uses: 40},
				"HD2_ReverbTest":   {Uses: 40},
			},
			want: 2,
		},
		{
			name: "two models used equally often, which break on identifier",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 90, Before: 90},
			},
			models: map[catalog.ModelID]corpus.ModelStats{
				"HD2_StereoDrive":  {Uses: 40},
				"HD2_DistMinotaur": {Uses: 40},
			},
			want:    1,
			wantIDs: []catalog.ModelID{"HD2_DistMinotaur"},
		},
		{
			// The corpus measures whatever presets held; a catalog for one
			// device will not carry all of it, and a chain cannot use what
			// the device lacks.
			name: "a model the catalog never heard of",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_GhostModel": {Uses: 900}},
		},
		{
			// It carries a slot index, not audio, so a generated chain
			// reaching for one would point at whatever happened to be loaded
			// there.
			name: "a block needing the owner's own IR",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryCab: {Chains: 95, After: 95},
			},
			models: map[catalog.ModelID]corpus.ModelStats{
				"HD2_ImpulseResponse1024": {Uses: 900},
			},
		},
		{
			// Line 6 tags amps and cabinets Guitar or Bass. This amp names no
			// cabinet, so the chain has a cabinet-shaped hole — and a guitar
			// cabinet must not fill it.
			name: "gear for the other instrument",
			gear: "Cabless Bass Head",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryCab: {Chains: 99, After: 99},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_CabGuitarOnly": {Uses: 900}},
		},
		{
			// Validation refuses a chain budgeted on an inferred cost.
			// Choosing one here would produce a chain rejected a moment
			// later, blaming a block nobody asked for.
			name: "a block whose cost was guessed",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryDrive: {Chains: 95, Before: 95},
			},
			models: map[catalog.ModelID]corpus.ModelStats{"HD2_NoDefault": {Uses: 900}},
		},
		{
			// Only amps and cabinets carry an instrument tag; a pedal serves
			// either. The guitar cabinet is refused, the pedal is not.
			name: "an untagged block, beside gear for the other instrument",
			gear: "Cabless Bass Head",
			cats: map[catalog.Category]corpus.CategoryStats{
				catalog.CategoryCab:   {Chains: 99, After: 99},
				catalog.CategoryDrive: {Chains: 80, Before: 80},
			},
			models: map[catalog.ModelID]corpus.ModelStats{
				"HD2_CabGuitarOnly": {Uses: 900},
				"HD2_DistMinotaur":  {Uses: 40},
			},
			want:    1,
			wantIDs: []catalog.ModelID{"HD2_DistMinotaur"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			gear := tt.gear
			if gear == "" {
				gear = "Ampeg SVT"
			}

			var stats *corpus.Stats

			switch {
			case tt.none:
			case tt.silent:
				stats = &corpus.Stats{Grammar: map[string]corpus.Grammar{}}
			default:
				stats = s.grammar(tt.cats, tt.models)
			}

			var first []catalog.ModelID

			for range 3 {
				spec, added, err := compile.Resolve(
					recipe(gear, "", tt.extra...), s.cat, stats)

				s.Require().NoError(err)
				s.Require().Len(added, tt.want)

				got := make([]catalog.ModelID, 0, len(added))
				for _, a := range added {
					got = append(got, a.Block.ID)
				}

				if first == nil {
					first = got
				}

				s.Require().Equal(first, got, "the same recipe must fill the same way")

				if tt.wantIDs != nil {
					s.Require().Equal(tt.wantIDs, got)
				}

				if tt.wantName != "" {
					s.Require().Equal(tt.wantName, added[0].Block.Name)
				}

				if tt.wantShare != 0 {
					s.Require().InDelta(tt.wantShare, added[0].Share, 1e-9)
				}

				if tt.first != "" {
					s.Require().Equal(tt.first, s.categoryAt(spec, 0))
				}

				if tt.last != "" {
					s.Require().Equal(
						tt.last, s.categoryAt(spec, len(spec.Blocks)-1))
				}
			}
		})
	}
}

// categoryAt returns the category of the block at a position.
func (s *FillPublicTestSuite) categoryAt(spec chain.Chain, i int) catalog.Category {
	b, ok := s.cat.Block(spec.Blocks[i].Model)
	s.Require().True(ok)

	return b.Category
}

func TestFillPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FillPublicTestSuite))
}
