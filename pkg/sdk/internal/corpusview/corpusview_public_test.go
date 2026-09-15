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
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusview/mocks"
)

type CorpusViewPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *CorpusViewPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// catalogs hands over the catalog at path, however often it is asked.
func (s *CorpusViewPublicTestSuite) catalogs(
	path string,
) *mocks.MockCatalogs {
	c := mocks.NewMockCatalogs(s.ctrl)
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.Open(path) },
	).AnyTimes()

	return c
}

// stats is the statistics at path, or this suite's fixture.
func stats(
	path string,
) string {
	if path == "" {
		return filepath.Join("testdata", "stats.json.gz")
	}

	return path
}

// TestModel covers what players did with one model.
//
// What it says about the measurements is the renderer's; this covers which
// model was asked about and what had to be opened to answer it.
func (s *CorpusViewPublicTestSuite) TestModel() {
	tests := []struct {
		name  string
		model string
		// files to read instead of this suite's own.
		stats   string
		catalog string
		err     error
		errText string
	}{
		{name: "one model's distributions", model: "HD2_AmpSVBeastNrm"},
		{
			// The corpus measures whatever presets contained; a catalog for
			// one device will not carry all of it. Answering with the
			// identifier beats pretending the model does not exist.
			name:  "a measured model the catalog never heard of",
			stats: filepath.Join("testdata", "bands.json.gz"),
			model: "HD2_GhostModel",
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
			// Not read as a question about chains instead: that is a
			// different question, asked of a different function.
			name: "no model at all",
			err:  corpusview.ErrNotMeasured,
		},
		{
			name:    "a statistics file that is not there",
			model:   "HD2_AmpSVBeastNrm",
			stats:   filepath.Join("testdata", "no.gz"),
			errText: "opening",
		},
		{
			name:    "a statistics file that is not gzip",
			model:   "HD2_AmpSVBeastNrm",
			stats:   filepath.Join("testdata", "notgzip.json.gz"),
			errText: "decoding",
		},
		{
			name:    "a catalog that is not there",
			model:   "HD2_AmpSVBeastNrm",
			catalog: filepath.Join("testdata", "no.json"),
			errText: "catalog",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cat := filepath.Join("testdata", "catalog.json")
			if tt.catalog != "" {
				cat = tt.catalog
			}

			measured, err := corpusview.Model(context.Background(), corpusview.Options{
				StatsPath: stats(tt.stats),
				Catalogs:  s.catalogs(cat),
			}, tt.model)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(measured.Stats)
			s.Require().True(measured.AboutOne())
			s.Require().Equal(tt.model, string(measured.Model))
			s.Require().NotNil(measured.Catalog)
			s.Require().Empty(measured.Instrument)
		})
	}
}

// TestChains covers what chains tend to hold.
func (s *CorpusViewPublicTestSuite) TestChains() {
	tests := []struct {
		name       string
		instrument string
		stats      string
		// the statistics this binary ships, rather than a fixture.
		builtIn bool
		errText string
	}{
		{name: "the grammar of every chain"},
		{name: "the grammar of one instrument", instrument: "guitar"},
		{name: "the statistics this binary ships", builtIn: true},
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// No expectation on the catalog: the grammar of a chain is block
			// kinds, so reaching for a catalog fails the row.
			o := corpusview.Options{
				StatsPath: stats(tt.stats),
				Catalogs:  mocks.NewMockCatalogs(s.ctrl),
			}
			if tt.builtIn {
				o.StatsPath = ""
			}

			measured, err := corpusview.Chains(o, tt.instrument)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(measured.Stats)
			s.Require().False(measured.AboutOne())
			s.Require().Equal(tt.instrument, measured.Instrument)
			s.Require().Nil(measured.Catalog)
		})
	}
}

func TestCorpusViewPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CorpusViewPublicTestSuite))
}
