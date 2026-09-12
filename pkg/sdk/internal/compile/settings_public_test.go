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
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
)

type SettingsPublicTestSuite struct {
	suite.Suite
	cat *catalog.Catalog
}

func (s *SettingsPublicTestSuite) SetupSuite() {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// stats builds statistics for the bass amp the fixture recipe resolves to.
func (s *SettingsPublicTestSuite) stats(
	params map[string]corpus.ParamStats,
) *corpus.Stats {
	return &corpus.Stats{
		Models: map[catalog.ModelID]corpus.ModelStats{
			"HD2_AmpSVBeastNrm": {Uses: 30, Params: params},
		},
	}
}

// value reads one parameter off the amp in a resolved chain.
func (s *SettingsPublicTestSuite) value(
	stats *corpus.Stats,
	key string,
) catalog.ParamValue {
	spec, _, _, err := compile.Resolve(recipe("Ampeg SVT (normal", ""), s.cat, stats)
	s.Require().NoError(err)
	s.Require().NotEmpty(spec.Blocks)

	return spec.Blocks[0].Params[key]
}

// TestResolveSettings covers what the corpus is allowed to say about a
// parameter. A nil want means the catalog's own default stands.
func (s *SettingsPublicTestSuite) TestResolveSettings() {
	tests := []struct {
		name    string
		params  map[string]corpus.ParamStats
		nothing bool
		unmet   bool
		key     string
		want    any
	}{
		{
			// Line 6 states one value; every player moved it to another and
			// they all agree. The middle of what people do is the better
			// answer.
			name: "close agreement, which overrules the catalog",
			params: map[string]corpus.ParamStats{
				"Treble": {N: 30, Median: 0.85, P25: 0.84, P75: 0.86},
			},
			key:  "Treble",
			want: 0.85,
		},
		{
			// An average of disagreement is not a measurement.
			name: "no agreement",
			params: map[string]corpus.ParamStats{
				"Drive": {N: 30, Median: 0.5, P25: 0.2, P75: 0.9},
			},
			key: "Drive",
		},
		{name: "no statistics at all", nothing: true, key: "Treble"},
		{
			name: "a parameter nobody measured",
			params: map[string]corpus.ParamStats{
				"SomethingElse": {N: 30, Median: 0.1, P25: 0.1, P75: 0.1},
			},
			key: "Treble",
		},
		{name: "a model nobody measured", unmet: true, key: "Treble"},
		{
			// A device given 1.5 for a three-position switch does not round
			// it, it refuses the preset. The median is rounded, not
			// truncated.
			name: "an integer parameter, which stays an integer",
			params: map[string]corpus.ParamStats{
				"MidFreq": {N: 30, Median: 1.6, P25: 1.6, P75: 1.6},
			},
			key:  "MidFreq",
			want: int64(2),
		},
		{
			// Truncation cuts toward zero, so a nudged -12.4 lands on -11
			// and an octave down becomes a major seventh.
			name: "an integer the corpus measured below nothing",
			params: map[string]corpus.ParamStats{
				"Interval": {N: 30, Median: -12.4, P25: -12.4, P75: -12.4},
			},
			key:  "Interval",
			want: int64(-12),
		},
		{
			// A median over a switch is not a setting the device will accept,
			// however unanimous the corpus is about it.
			name: "a switch, which is never averaged",
			params: map[string]corpus.ParamStats{
				"Bright": {N: 30, Median: 1, P25: 1, P75: 1},
			},
			key:  "Bright",
			want: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var stats *corpus.Stats

			switch {
			case tt.nothing:
			case tt.unmet:
				stats = &corpus.Stats{
					Models: map[catalog.ModelID]corpus.ModelStats{},
				}
			default:
				stats = s.stats(tt.params)
			}

			got := s.value(stats, tt.key)

			switch want := tt.want.(type) {
			case nil:
				def, ok := s.cat.
					Blocks["HD2_AmpSVBeastNrm"].Params[tt.key].Default.Float()
				s.Require().True(ok)

				f, ok := got.Float()
				s.Require().True(ok)
				s.Require().InDelta(def, f, 1e-9)
			case float64:
				f, ok := got.Float()
				s.Require().True(ok)
				s.Require().InDelta(want, f, 1e-9)
			case int64:
				s.Require().Equal(catalog.ParamInt, got.Type())

				i, ok := got.Int()
				s.Require().True(ok)
				s.Require().Equal(want, i)
			case bool:
				s.Require().Equal(catalog.ParamBool, got.Type())

				b, ok := got.Bool()
				s.Require().True(ok)
				s.Require().Equal(want, b)
			}
		})
	}
}

func TestSettingsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SettingsPublicTestSuite))
}
