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

// TestShow reads what the corpus measured.
//
// What it says about the measurements is the renderer's; this covers which
// question was asked and what had to be opened to answer it.
func (s *CorpusViewPublicTestSuite) TestShow() {
	tests := []struct {
		name       string
		model      string
		instrument string
		// files to read instead of this suite's own.
		stats   string
		catalog string
		// the statistics this binary ships, rather than a fixture.
		builtIn bool

		// a model was asked about, so the catalog comes with the answer.
		aboutOne bool
		err      error
		errText  string
	}{
		{
			name:     "one model's distributions",
			model:    "HD2_AmpSVBeastNrm",
			aboutOne: true,
		},
		{
			// The corpus measures whatever presets contained; a catalog for
			// one device will not carry all of it. Answering with the
			// identifier beats pretending the model does not exist.
			name:     "a measured model the catalog never heard of",
			stats:    filepath.Join("testdata", "bands.json.gz"),
			model:    "HD2_GhostModel",
			aboutOne: true,
		},
		{
			name: "the grammar of a chain",
		},
		{
			name:       "the grammar of one instrument",
			instrument: "guitar",
		},
		{
			name:    "the statistics this binary ships",
			builtIn: true,
		},
		{
			// Refused by the operation rather than drawn as a table with
			// nothing in it.
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
			// Only a model needs one, so this is the one shape where a
			// missing catalog is a failure.
			name:    "a catalog that is not there",
			model:   "HD2_AmpSVBeastNrm",
			catalog: filepath.Join("testdata", "no.json"),
			errText: "catalog",
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

			measured, err := corpusview.Show(o)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(measured.Stats)
			s.Require().Equal(tt.aboutOne, measured.AboutOne())
			s.Require().Equal(tt.instrument, measured.Instrument)

			if tt.aboutOne {
				s.Require().Equal(tt.model, string(measured.Model))
				s.Require().NotNil(measured.Catalog)

				return
			}

			s.Require().Nil(measured.Catalog)
		})
	}
}

func TestCorpusViewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusViewPublicTestSuite))
}
