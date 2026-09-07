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
	"github.com/retr0h/tonestack/pkg/corpus"
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
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT (normal", ""), s.cat, stats)
	s.Require().NoError(err)
	s.Require().NotEmpty(spec.Blocks)

	return spec.Blocks[0].Params[key]
}

func (s *SettingsPublicTestSuite) TestCloseAgreementOverrulesTheCatalog() {
	// Line 6 states one value; every player moved it to another and they all
	// agree. The middle of what people do is the better answer.
	got := s.value(s.stats(map[string]corpus.ParamStats{
		"Treble": {N: 30, Median: 0.85, P25: 0.84, P75: 0.86},
	}), "Treble")

	f, ok := got.Float()
	s.Require().True(ok)
	s.Require().InDelta(0.85, f, 1e-9)
}

func (s *SettingsPublicTestSuite) TestNoAgreementLeavesTheCatalogAlone() {
	def, _ := s.cat.Blocks["HD2_AmpSVBeastNrm"].Params["Drive"].Default.Float()

	got := s.value(s.stats(map[string]corpus.ParamStats{
		"Drive": {N: 30, Median: 0.5, P25: 0.2, P75: 0.9},
	}), "Drive")

	f, _ := got.Float()
	s.Require().InDelta(def, f, 1e-9,
		"an average of disagreement is not a measurement")
}

func (s *SettingsPublicTestSuite) TestWithoutStatisticsTheCatalogStands() {
	def, _ := s.cat.Blocks["HD2_AmpSVBeastNrm"].Params["Treble"].Default.Float()

	f, _ := s.value(nil, "Treble").Float()

	s.Require().InDelta(def, f, 1e-9)
}

func (s *SettingsPublicTestSuite) TestAParameterNobodyMeasuredKeepsItsDefault() {
	def, _ := s.cat.Blocks["HD2_AmpSVBeastNrm"].Params["Treble"].Default.Float()

	f, _ := s.value(s.stats(map[string]corpus.ParamStats{
		"SomethingElse": {N: 30, Median: 0.1, P25: 0.1, P75: 0.1},
	}), "Treble").Float()

	s.Require().InDelta(def, f, 1e-9)
}

func (s *SettingsPublicTestSuite) TestAModelNobodyMeasuredKeepsItsDefaults() {
	def, _ := s.cat.Blocks["HD2_AmpSVBeastNrm"].Params["Treble"].Default.Float()

	f, _ := s.value(&corpus.Stats{
		Models: map[catalog.ModelID]corpus.ModelStats{},
	}, "Treble").Float()

	s.Require().InDelta(def, f, 1e-9)
}

func (s *SettingsPublicTestSuite) TestAnIntegerParameterStaysAnInteger() {
	// A device given 1.5 for a three-position switch does not round it, it
	// refuses the preset.
	got := s.value(s.stats(map[string]corpus.ParamStats{
		"MidFreq": {N: 30, Median: 1.6, P25: 1.6, P75: 1.6},
	}), "MidFreq")

	s.Require().Equal(catalog.ParamInt, got.Type())

	i, ok := got.Int()
	s.Require().True(ok)
	s.Require().Equal(int64(2), i, "the median is rounded, not truncated")
}

func (s *SettingsPublicTestSuite) TestASwitchIsNeverAveraged() {
	// A median over a switch is not a setting the device will accept, however
	// unanimous the corpus is about it.
	got := s.value(s.stats(map[string]corpus.ParamStats{
		"Bright": {N: 30, Median: 1, P25: 1, P75: 1},
	}), "Bright")

	s.Require().Equal(catalog.ParamBool, got.Type())

	b, ok := got.Bool()
	s.Require().True(ok)
	s.Require().False(b, "the catalog's own default stands")
}

func TestSettingsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SettingsPublicTestSuite))
}
