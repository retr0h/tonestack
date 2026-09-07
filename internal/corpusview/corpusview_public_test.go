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

package corpusview_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/corpusview"
)

type CorpusViewPublicTestSuite struct {
	suite.Suite
}

func (s *CorpusViewPublicTestSuite) opts() corpusview.Options {
	return corpusview.Options{
		StatsPath:   filepath.Join("testdata", "stats.json.gz"),
		CatalogPath: filepath.Join("testdata", "catalog.json"),
	}
}

func (s *CorpusViewPublicTestSuite) TestShowsOneModelsDistributions() {
	o := s.opts()
	o.Model = "HD2_AmpSVBeastNrm"

	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, o))

	got := out.String()
	s.Require().Contains(got, "Ampeg SVT Nrm", "the model is named, not just identified")
	s.Require().Contains(got, "Drive")
	s.Require().Contains(got, "0.440", "the median is what people actually set")
	s.Require().Contains(got, "unanimous", "a parameter nobody varies says so")
}

func (s *CorpusViewPublicTestSuite) TestShowsChainGrammar() {
	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, s.opts()))

	got := out.String()
	s.Require().Contains(got, "bass")
	s.Require().Contains(got, "drive")
	s.Require().Contains(got, "100%", "every drive in this corpus precedes the amp")
}

func (s *CorpusViewPublicTestSuite) TestGrammarCanBeLimitedToOneInstrument() {
	o := s.opts()
	o.Instrument = "guitar"

	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, o))

	s.Require().Contains(out.String(), "nothing measured")
}

func (s *CorpusViewPublicTestSuite) TestShowsAModelTheCatalogDoesNotName() {
	o := s.opts()
	o.Model = "HD2_Cab8x10SVBeast"
	o.CatalogPath = filepath.Join("testdata", "empty-catalog.json")

	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, o))

	s.Require().Contains(out.String(), "HD2_Cab8x10SVBeast",
		"an unnamed model is shown by identifier rather than hidden")
}

func (s *CorpusViewPublicTestSuite) TestAgreementIsGradedAgainstTheParametersRange() {
	o := s.opts()
	o.StatsPath = filepath.Join("testdata", "bands.json.gz")
	o.Model = "HD2_AmpSVBeastNrm"

	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, o))

	got := out.String()
	for _, want := range []string{"unanimous", "close", "loose", "none"} {
		s.Require().Contains(got, want,
			"each band must be reachable, or the grading says nothing")
	}
	// A parameter the catalog does not carry has no range to grade against.
	s.Require().Contains(got, "Ghost")
}

func (s *CorpusViewPublicTestSuite) TestTiedCategoriesAreOrderedStably() {
	o := s.opts()
	o.StatsPath = filepath.Join("testdata", "bands.json.gz")

	var first string

	for range 3 {
		var out bytes.Buffer
		s.Require().NoError(corpusview.Show(&out, o))

		if first == "" {
			first = out.String()
		}

		s.Require().Equal(first, out.String(),
			"two categories used equally often must not shuffle between runs")
	}
}

func (s *CorpusViewPublicTestSuite) TestFallsBackToTheBuiltInStatistics() {
	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, corpusview.Options{}))

	s.Require().Contains(out.String(), "guitar")
}

func (s *CorpusViewPublicTestSuite) TestAMeasuredModelTheCatalogNeverHeardOf() {
	// The corpus measures whatever presets contained; a catalog for one
	// device will not carry all of it. Showing the identifier is more useful
	// than pretending the model does not exist.
	o := s.opts()
	o.StatsPath = filepath.Join("testdata", "bands.json.gz")
	o.Model = "HD2_GhostModel"

	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, o))

	got := out.String()
	s.Require().Contains(got, "HD2_GhostModel")
	s.Require().Contains(got, "Drive")
}

func (s *CorpusViewPublicTestSuite) TestReportsProblems() {
	tests := []struct {
		name    string
		mutate  func(*corpusview.Options)
		want    error
		message string
	}{
		{
			"a model nobody used",
			func(o *corpusview.Options) { o.Model = "HD2_NoSuchModel" },
			corpusview.ErrNotMeasured, "no preset in the corpus uses",
		},
		{
			"a statistics file that is not there",
			func(o *corpusview.Options) { o.StatsPath = filepath.Join("testdata", "no.gz") },
			nil, "opening",
		},
		{
			"a statistics file that is not gzip",
			func(o *corpusview.Options) {
				o.StatsPath = filepath.Join("testdata", "notgzip.json.gz")
			},
			nil, "decoding",
		},
		{
			"a catalog that is not there",
			func(o *corpusview.Options) {
				o.Model = "HD2_AmpSVBeastNrm"
				o.CatalogPath = filepath.Join("testdata", "no.json")
			},
			nil, "catalog",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			o := s.opts()
			tc.mutate(&o)

			err := corpusview.Show(&bytes.Buffer{}, o)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)

			if tc.want != nil {
				s.Require().ErrorIs(err, tc.want)
			}
		})
	}
}

func (s *CorpusViewPublicTestSuite) TestReportsAWriterThatFails() {
	o := s.opts()

	s.Require().Error(corpusview.Show(&failingWriter{}, o))

	o.Model = "HD2_AmpSVBeastNrm"
	s.Require().Error(corpusview.Show(&failingWriter{}, o))
}

func (s *CorpusViewPublicTestSuite) TestSaysWhenNothingWasMeasured() {
	o := s.opts()
	o.StatsPath = filepath.Join("testdata", "empty.json.gz")

	var out bytes.Buffer
	s.Require().NoError(corpusview.Show(&out, o))

	s.Require().Contains(out.String(), "nothing measured")
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestCorpusViewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusViewPublicTestSuite))
}
