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
	"io"
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

// TestShow prints what the corpus measured.
func (s *CorpusViewPublicTestSuite) TestShow() {
	tests := []struct {
		name       string
		model      string
		instrument string
		// files to read instead of this suite's own.
		stats   string
		catalog string
		// the statistics and catalog this binary ships, rather than fixtures.
		builtIn bool
		deaf    bool
		// show it three times, for an ordering that must not shuffle.
		stable bool

		contains []string
		err      error
		errText  string
	}{
		{
			name:  "one model's distributions",
			model: "HD2_AmpSVBeastNrm",
			contains: []string{
				// The model is named, not just identified.
				"Ampeg SVT Nrm",
				"Drive",
				// The median is what people actually set.
				"0.440",
				// A parameter nobody varies says so.
				"unanimous",
			},
		},
		{
			name: "the grammar of a chain",
			// Every drive in this corpus precedes the amp.
			contains: []string{"bass", "drive", "100%"},
		},
		{
			name:       "an instrument nobody measured",
			instrument: "guitar",
			contains:   []string{"nothing measured"},
		},
		{
			// An unnamed model is shown by identifier rather than hidden.
			name:     "a model the catalog does not name",
			model:    "HD2_Cab8x10SVBeast",
			catalog:  filepath.Join("testdata", "empty-catalog.json"),
			contains: []string{"HD2_Cab8x10SVBeast"},
		},
		{
			name:  "agreement, graded against each parameter's range",
			stats: filepath.Join("testdata", "bands.json.gz"),
			model: "HD2_AmpSVBeastNrm",
			contains: []string{
				// Each band must be reachable, or the grading says nothing.
				"unanimous", "close", "loose", "none",
				// A parameter the catalog does not carry has no range to
				// grade against.
				"Ghost",
			},
		},
		{
			// Two categories used equally often must not shuffle between
			// runs.
			name:   "categories used equally often",
			stats:  filepath.Join("testdata", "bands.json.gz"),
			stable: true,
		},
		{
			name:     "the statistics this binary ships",
			builtIn:  true,
			contains: []string{"guitar"},
		},
		{
			// The corpus measures whatever presets contained; a catalog for
			// one device will not carry all of it. Showing the identifier is
			// more useful than pretending the model does not exist.
			name:     "a measured model the catalog never heard of",
			stats:    filepath.Join("testdata", "bands.json.gz"),
			model:    "HD2_GhostModel",
			contains: []string{"HD2_GhostModel", "Drive"},
		},
		{
			name:     "a corpus nobody measured anything from",
			stats:    filepath.Join("testdata", "empty.json.gz"),
			contains: []string{"nothing measured"},
		},
		{
			name:    "a model nobody used",
			model:   "HD2_NoSuchModel",
			err:     corpusview.ErrNotMeasured,
			errText: "no preset in the corpus uses",
		},
		{
			name:    "a statistics file that is not there",
			stats:   filepath.Join("testdata", "no.gz"),
			errText: "opening",
		},
		{
			name:    "a statistics file that is not gzip",
			stats:   filepath.Join("testdata", "notgzip.json.gz"),
			errText: "decoding",
		},
		{
			name:    "a catalog that is not there",
			model:   "HD2_AmpSVBeastNrm",
			catalog: filepath.Join("testdata", "no.json"),
			errText: "catalog",
		},
		{name: "a writer that fails on the grammar", deaf: true},
		{
			name:  "a writer that fails on a model",
			model: "HD2_AmpSVBeastNrm",
			deaf:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			o := s.opts()
			if tt.builtIn {
				o = corpusview.Options{}
			}

			o.Model = tt.model
			o.Instrument = tt.instrument

			if tt.stats != "" {
				o.StatsPath = tt.stats
			}

			if tt.catalog != "" {
				o.CatalogPath = tt.catalog
			}

			if tt.deaf {
				s.Require().Error(corpusview.Show(&failingWriter{}, o))

				return
			}

			var first string

			runs := 1
			if tt.stable {
				runs = 3
			}

			for range runs {
				var out bytes.Buffer

				err := corpusview.Show(io.Writer(&out), o)

				if tt.err != nil || tt.errText != "" {
					s.Require().Error(err)

					if tt.err != nil {
						s.Require().ErrorIs(err, tt.err)
					}

					s.Require().Contains(err.Error(), tt.errText)

					return
				}

				s.Require().NoError(err)

				if first == "" {
					first = out.String()
				}

				s.Require().Equal(first, out.String(),
					"the same corpus must read the same way every run")

				for _, want := range tt.contains {
					s.Require().Contains(out.String(), want)
				}
			}
		})
	}
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestCorpusViewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusViewPublicTestSuite))
}
