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
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusgen"
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

// partLocked returns a directory holding one readable preset and one nobody
// can open.
func (s *CorpusgenPublicTestSuite) partLocked() string {
	dir := s.T().TempDir()

	src, err := os.ReadFile(filepath.Join("testdata", "corpus", "bass0.hlx"))
	s.Require().NoError(err)

	s.Require().NoError(os.WriteFile(filepath.Join(dir, "ok.hlx"), src, 0o600))

	locked := filepath.Join(dir, "locked.hlx")
	s.Require().NoError(os.WriteFile(locked, src, 0o600))
	s.Require().NoError(os.Chmod(locked, 0o000))

	return dir
}

// TestRun measures a body of presets other people made.
func (s *CorpusgenPublicTestSuite) TestRun() {
	tests := []struct {
		name string
		// which corpus to read: this suite's fixture unless a case says
		// otherwise.
		corpus     string
		catalog    string
		out        string
		minSamples int

		// what the measurement must say about one parameter of the amp.
		param      string
		wantN      int
		wantMedian float64
		// parameters of that amp which must not be recorded at all.
		absent []string
		// what the grammar must say.
		chains     int
		before     map[catalog.Category]float64
		noCategory []catalog.Category
		// how many chains for the instrument held each model.
		models   map[catalog.ModelID]int
		grammars int
		presets  int

		err     error
		errText string
	}{
		{
			name: "a corpus of presets",

			// Eleven presets sit between 0.40 and 0.47, two outliers at 0.95,
			// and one belongs to another device entirely. The median ignores
			// the outliers; the spread records that they exist. An Ampeg SVT
			// is the same model with the same controls whichever Helix
			// carries it, so excluding that preset would only shrink the
			// sample.
			param:      "Drive",
			wantN:      14,
			wantMedian: 0.44,

			// A median over an enumeration is meaningless, and the average of
			// a switch is a value the device will not accept.
			absent: []string{"Bright", "Voicing"},

			// A chain with no amp says nothing about ordering and is not
			// counted.
			chains: 14,
			before: map[catalog.Category]float64{
				catalog.CategoryDrive: 1,
				catalog.CategoryCab:   0,
			},
			// A volume block in most chains says nothing about how a tone is
			// built. And Line 6 tag amps Guitar or Bass; a mic preamp is
			// tagged neither, so a chain built around one belongs to no
			// instrument's habits.
			noCategory: []catalog.Category{catalog.CategoryUtility},
			// Counted per instrument, once per chain. The Minotaur sits in
			// eight bass chains and in two presets that belong to no
			// instrument, one with no amp and one built around a preamp;
			// those two are not counted.
			models: map[catalog.ModelID]int{
				"HD2_AmpSVBeastNrm": 14,
				"HD2_DistMinotaur":  8,
				"HD2_EqTest":        2,
			},
			grammars: 1,
		},
		{
			// A median over too few presets is an anecdote with a decimal
			// point.
			name:       "parameters seen too rarely to mean anything",
			minSamples: 99,
			absent:     []string{"Drive"},
		},
		{
			// One unreadable file among thousands is a fact about that file,
			// not a reason to abandon the measurement.
			name:       "a preset nobody can open",
			corpus:     "part-locked",
			minSamples: 1,
			presets:    1,
		},
		{
			name:    "a corpus directory that is not there",
			corpus:  "missing",
			errText: "searching",
		},
		{
			name:    "a directory holding no presets",
			corpus:  "empty",
			err:     corpusgen.ErrNoPresets,
			errText: "no presets",
		},
		{
			name:    "a catalog that is not there",
			catalog: filepath.Join("testdata", "no.json"),
			errText: "catalog",
		},
		{
			name:    "a destination directory that is not there",
			out:     filepath.Join("no", "s.gz"),
			errText: "writing",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "s.gz")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			o := s.opts(out)
			o.MinSamples = tt.minSamples

			switch tt.corpus {
			case "":
			case "missing":
				o.CorpusDir = filepath.Join("testdata", "nope")
			case "empty":
				o.CorpusDir = s.T().TempDir()
			case "part-locked":
				o.CorpusDir = s.partLocked()
			}

			if tt.catalog != "" {
				o.CatalogPath = tt.catalog
			}

			counted, err := corpusgen.Run(o)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().FileExists(out)
			s.Require().Equal(out, counted.Path)

			// The answer carries the measurements, so nothing has to read
			// the file back to find out what was written.
			s.Require().Equal(s.read(out).Presets, counted.Stats.Presets)

			stats := s.read(out)

			if tt.param != "" {
				p, ok := stats.Param("HD2_AmpSVBeastNrm", tt.param)

				s.Require().True(ok)
				s.Require().Equal(tt.wantN, p.N)
				s.Require().InDelta(tt.wantMedian, p.Median, 1e-9)
				s.Require().Positive(p.Spread())
			}

			for _, key := range tt.absent {
				_, ok := stats.Param("HD2_AmpSVBeastNrm", key)

				s.Require().False(ok, "%s must not be recorded", key)
			}

			if tt.chains > 0 {
				g, ok := stats.Grammar["bass"]

				s.Require().True(ok)
				s.Require().Equal(tt.chains, g.Chains)

				for cat, want := range tt.before {
					s.Require().InDelta(want, g.Categories[cat].BeforeAmp(), 1e-9,
						"where a %s sits", cat)
				}

				for _, cat := range tt.noCategory {
					s.Require().NotContains(g.Categories, cat)
				}

				for id, want := range tt.models {
					s.Require().Equal(want, g.Models[id], "chains holding %s", id)
				}
			}

			if tt.grammars > 0 {
				s.Require().Len(stats.Grammar, tt.grammars)
			}

			if tt.presets > 0 {
				s.Require().Equal(tt.presets, stats.Presets)
			}
		})
	}
}

// read loads statistics this suite generated.
func (s *CorpusgenPublicTestSuite) read(path string) *corpus.Stats {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	zr, err := gzip.NewReader(f)
	s.Require().NoError(err)

	stats, err := corpus.Load(zr)
	s.Require().NoError(err)

	return stats
}

func TestCorpusgenPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusgenPublicTestSuite))
}
