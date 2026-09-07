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

package corpusgen_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/corpusgen"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/corpus"
)

type CorpusgenPublicTestSuite struct {
	suite.Suite
}

func (s *CorpusgenPublicTestSuite) opts(out string) corpusgen.Options {
	return corpusgen.Options{
		CorpusDir:   filepath.Join("testdata", "corpus"),
		CatalogPath: filepath.Join("testdata", "catalog.json"),
		OutputPath:  out,
	}
}

func (s *CorpusgenPublicTestSuite) TestRunMeasuresACorpus() {
	out := filepath.Join(s.T().TempDir(), "stats.json.gz")

	var log bytes.Buffer
	s.Require().NoError(corpusgen.Run(&log, s.opts(out)))

	s.Require().FileExists(out)
	s.Require().Contains(log.String(), "presets measured")
	s.Require().Contains(log.String(), "bass")
}

func (s *CorpusgenPublicTestSuite) TestMeasuredValuesAreTheMiddleOfWhatPeopleDo() {
	stats := s.measure(s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	p, ok := stats.Param("HD2_AmpSVBeastNrm", "Drive")

	s.Require().True(ok)
	s.Require().Equal(14, p.N)
	// Eleven presets sit between 0.40 and 0.47, two outliers at 0.95, and one
	// belongs to another device entirely. The median ignores the outliers;
	// the spread records that they exist.
	s.Require().InDelta(0.44, p.Median, 1e-9)
	s.Require().Positive(p.Spread())
}

func (s *CorpusgenPublicTestSuite) TestPresetsForOtherDevicesAreMeasuredToo() {
	// An Ampeg SVT is the same model with the same controls whichever Helix
	// carries it, so excluding those presets would only shrink the sample.
	stats := s.measure(s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	p, _ := stats.Param("HD2_AmpSVBeastNrm", "Drive")

	s.Require().Equal(14, p.N,
		"every preset using this amp counts, including one for another device")
}

func (s *CorpusgenPublicTestSuite) TestGrammarRecordsWhereBlocksSit() {
	stats := s.measure(s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	g, ok := stats.Grammar["bass"]

	s.Require().True(ok)
	s.Require().Equal(14, g.Chains,
		"a chain with no amp says nothing about ordering and is not counted")
	s.Require().InDelta(1.0, g.Categories["drive"].BeforeAmp(), 1e-9,
		"every drive in the fixture sits ahead of the amp")
	s.Require().Zero(g.Categories["cab"].BeforeAmp(),
		"a cabinet belongs after the amp")
}

func (s *CorpusgenPublicTestSuite) TestSwitchesAreNotAveraged() {
	// A median over an enumeration is meaningless, and the average of a
	// switch is a value the device will not accept.
	stats := s.measure(s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	for _, key := range []string{"Bright", "Voicing"} {
		s.Run(key, func() {
			_, ok := stats.Param("HD2_AmpSVBeastNrm", key)

			s.Require().False(ok)
		})
	}
}

func (s *CorpusgenPublicTestSuite) TestPlumbingIsNotPartOfTheGrammar() {
	// A volume block in most chains says nothing about how a tone is built.
	stats := s.measure(s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	s.Require().NotContains(stats.Grammar["bass"].Categories, catalog.CategoryUtility)
}

func (s *CorpusgenPublicTestSuite) TestAnAmpForNeitherInstrumentIsSkipped() {
	// Line 6 tags amps Guitar or Bass; a mic preamp is tagged neither, and a
	// chain built around one belongs to no instrument's habits.
	stats := s.measure(s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	s.Require().NotContains(stats.Grammar, "preamp > mic")
	s.Require().Len(stats.Grammar, 1, "only bass chains are in this fixture")
}

func (s *CorpusgenPublicTestSuite) TestParametersSeenTooRarelyAreNotRecorded() {
	o := s.opts(filepath.Join(s.T().TempDir(), "s.gz"))
	o.MinSamples = 99
	stats := s.measure(o)

	_, ok := stats.Param("HD2_AmpSVBeastNrm", "Drive")

	s.Require().False(ok,
		"a median over too few presets is an anecdote with a decimal point")
}

func (s *CorpusgenPublicTestSuite) TestAPresetThatCannotBeOpenedIsSkipped() {
	// One unreadable file among thousands is a fact about that file, not a
	// reason to abandon the measurement.
	dir := s.T().TempDir()

	src, err := os.ReadFile(filepath.Join("testdata", "corpus", "bass0.hlx"))
	s.Require().NoError(err)

	s.Require().NoError(os.WriteFile(filepath.Join(dir, "ok.hlx"), src, 0o600))

	locked := filepath.Join(dir, "locked.hlx")
	s.Require().NoError(os.WriteFile(locked, src, 0o600))
	s.Require().NoError(os.Chmod(locked, 0o000))

	o := s.opts(filepath.Join(s.T().TempDir(), "s.gz"))
	o.CorpusDir = dir
	o.MinSamples = 1

	stats := s.measure(o)

	s.Require().Equal(1, stats.Presets, "the readable one was still measured")
}

func (s *CorpusgenPublicTestSuite) TestRunReportsProblems() {
	dir := s.T().TempDir()

	tests := []struct {
		name    string
		mutate  func(*corpusgen.Options)
		want    error
		message string
	}{
		{
			"a corpus directory that is not there",
			func(o *corpusgen.Options) { o.CorpusDir = filepath.Join("testdata", "nope") },
			nil, "searching",
		},
		{
			"a directory holding no presets",
			func(o *corpusgen.Options) { o.CorpusDir = dir },
			corpusgen.ErrNoPresets, "no presets",
		},
		{
			"a catalog that is not there",
			func(o *corpusgen.Options) { o.CatalogPath = filepath.Join("testdata", "no.json") },
			nil, "catalog",
		},
		{
			"a destination directory that is not there",
			func(o *corpusgen.Options) { o.OutputPath = filepath.Join(dir, "no", "s.gz") },
			nil, "writing",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			o := s.opts(filepath.Join(dir, "s.gz"))
			tc.mutate(&o)

			err := corpusgen.Run(&bytes.Buffer{}, o)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)

			if tc.want != nil {
				s.Require().ErrorIs(err, tc.want)
			}
		})
	}
}

func (s *CorpusgenPublicTestSuite) TestRunReportsAWriterThatFails() {
	err := corpusgen.Run(&failingWriter{},
		s.opts(filepath.Join(s.T().TempDir(), "s.gz")))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reporting")
}

// measure runs a generation and reads the result back.
func (s *CorpusgenPublicTestSuite) measure(o corpusgen.Options) *corpus.Stats {
	s.Require().NoError(corpusgen.Run(&bytes.Buffer{}, o))

	f, err := os.Open(o.OutputPath) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	zr, err := gzip.NewReader(f)
	s.Require().NoError(err)

	stats, err := corpus.Load(zr)
	s.Require().NoError(err)

	return stats
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestCorpusgenPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusgenPublicTestSuite))
}
