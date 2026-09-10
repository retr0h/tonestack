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

package cli_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	gen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

type RecipePublicTestSuite struct {
	suite.Suite
}

// rig builds a rig carrying everything a person can write down, so a test
// about rendering one is not also a test about what a rig must hold.
func rigWith(mutate func(*gen.RigSpec)) gen.RigSpec {
	v := gen.RigSpecVersion(2)
	band := "Green Day"
	era := "Dookie through American Idiot"
	conf := gen.ConfidenceHigh
	pos := gen.PositionBridge
	mute := gen.MutingPalm
	cited := []gen.Evidence{{Kind: gen.EvidenceCited}}

	spec := gen.RigSpec{
		Schema:     "RigSpec",
		Version:    &v,
		ID:         "mike-dirnt",
		Subject:    gen.Subject{Kind: "artist", Name: "Mike Dirnt", Band: &band, Era: &era},
		Instrument: "bass",
		// Confirmed by default, so a rig that says nobody checked it is a
		// case a test has to ask for rather than get by accident.
		Chain: []gen.ChainEntry{
			{Gear: "Ampeg SVT", Role: gen.RoleAmp, Evidence: &cited},
			{Gear: "Ampeg 8x10", Role: gen.RoleCab, Evidence: &cited},
		},
		Character:  &[]gen.CharacterTerm{{Term: "mid-forward"}, {Term: "gritty"}},
		Confidence: &conf,
		Technique: &gen.Technique{
			Attack:   gen.AttackPick,
			Position: &pos,
			Muting:   &mute,
		},
	}

	if mutate != nil {
		mutate(&spec)
	}

	return spec
}

// TestRecipes covers listing every rig a directory holds.
func (s *RecipePublicTestSuite) TestRecipes() {
	tests := []struct {
		name string
		in   sdk.Recipes
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "one row per recipe",
			in: sdk.Recipes{
				Dir:  "pkg/sdk/rigs",
				Rigs: []gen.RigSpec{rigWith(nil)},
			},
			want: []string{"mike-dirnt", "Mike Dirnt", "bass", "Ampeg SVT"},
		},
		{
			// The source column is the one that decides whether to trust the
			// row, so a rig nobody confirmed has to read differently from
			// one somebody did.
			name: "a recipe nobody confirmed",
			in: sdk.Recipes{
				Dir: "pkg/sdk/rigs",
				Rigs: []gen.RigSpec{rigWith(func(r *gen.RigSpec) {
					r.Chain[0].Evidence = nil
				})},
			},
			want: []string{"mike-dirnt"},
		},
		{
			name: "a shelf with nothing on it",
			in:   sdk.Recipes{Dir: "pkg/sdk/rigs"},
			want: []string{"no recipes here"},
		},
		{
			name: "nowhere to write it",
			in: sdk.Recipes{
				Dir:  "pkg/sdk/rigs",
				Rigs: []gen.RigSpec{rigWith(nil)},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to write the empty case either",
			in:   sdk.Recipes{Dir: "pkg/sdk/rigs"},
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Recipes(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestRecipe covers printing one rig in full.
func (s *RecipePublicTestSuite) TestRecipe() {
	tests := []struct {
		name   string
		in     sdk.Recipe
		to     io.Writer
		want   []string
		absent []string
		err    bool
	}{
		{
			name: "everything a person wrote",
			in: sdk.Recipe{
				Rig: rigWith(nil),
				Variants: []sdk.Variant{
					{ID: "mike-dirnt-longview", Name: "Longview"},
				},
			},
			want: []string{
				"Mike Dirnt", "Green Day", "Dookie through American Idiot",
				// Three stored fields, said as the one sentence a person would.
				"pick, near the bridge, palm muted",
				"mid-forward", "variants", "Longview (mike-dirnt-longview)",
				"high confidence",
			},
		},
		{
			// Nothing about what nobody wrote. An empty band line reads as a
			// band with no name rather than as a player without one.
			name: "a rig with only the required fields",
			in: sdk.Recipe{Rig: rigWith(func(r *gen.RigSpec) {
				r.Subject.Band = nil
				r.Subject.Era = nil
				r.Character = nil
				r.Technique = nil
			})},
			absent: []string{"character", "variants", "band", "era", "technique"},
		},
		{
			// An unstated confidence is the lowest one. A rig that says
			// nothing about how far to trust it has not earned anything.
			name: "an unstated confidence reads as low",
			in: sdk.Recipe{Rig: rigWith(func(r *gen.RigSpec) {
				r.Confidence = nil
			})},
			want: []string{"low confidence"},
		},
		{
			// A rig nobody has confirmed says so, whatever it claims about
			// itself.
			name: "gear nobody confirmed",
			in: sdk.Recipe{Rig: rigWith(func(r *gen.RigSpec) {
				r.Chain[0].Evidence = nil
			})},
			want: []string{"unverified"},
		},
		{
			name: "nowhere to write it",
			in:   sdk.Recipe{Rig: rigWith(nil)},
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Recipe(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

// TestScaffolded covers saying what recipe was written.
func (s *RecipePublicTestSuite) TestScaffolded() {
	full := sdk.Scaffolded{
		ID:         "test-player",
		Name:       "Test Player",
		Instrument: "bass",
		Amp:        "Ampeg SVT",
		Cab:        "Ampeg 8x10",
		Pedals:     []string{"Klon Centaur", "Boss DS-1"},
		Path:       "pkg/sdk/rigs/artists/test-player.yaml",
	}

	tests := []struct {
		name   string
		in     sdk.Scaffolded
		to     io.Writer
		want   []string
		absent []string
		err    bool
	}{
		{
			name: "everything it was given",
			in:   full,
			want: []string{
				"Test Player", "test-player", "Ampeg SVT", "Ampeg 8x10",
				"Klon Centaur, Boss DS-1",
				// What to do with it next, which is the point of saying
				// anything at all.
				"presets make --id test-player",
			},
		},
		{
			// Some amps carry their own cabinet, and a blank row would read
			// as a cabinet nobody named.
			name: "an amp that carries its own cabinet",
			in: sdk.Scaffolded{
				ID: "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "Ampeg SVT",
				Path: "x.yaml",
			},
			absent: []string{"cab", "pedals"},
		},
		{
			name: "nowhere to say it",
			in:   full,
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Scaffolded(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

func TestRecipePublicTestSuite(t *testing.T) {
	suite.Run(t, new(RecipePublicTestSuite))
}
