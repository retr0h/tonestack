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
package catalogview_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/catalogview"
)

type CatalogViewPublicTestSuite struct {
	suite.Suite
}

func (s *CatalogViewPublicTestSuite) path() string {
	return filepath.Join("testdata", "catalog.json")
}

// TestOpen reads a catalog off disk, or the one built into the binary.
func (s *CatalogViewPublicTestSuite) TestOpen() {
	tests := []struct {
		name   string
		path   string
		device string
		blocks int
		err    bool
		says   string
	}{
		{
			name:   "a catalog somebody generated",
			path:   s.path(),
			device: "HX Stomp",
			blocks: 2,
		},
		{
			// No path is the case for anyone who has not generated their
			// own, which is everyone who installed a binary.
			name: "no path falls back to the built-in one",
			path: "",
		},
		{
			name: "a file that is not there",
			path: filepath.Join("testdata", "nope.json"),
			err:  true,
			says: "opening catalog",
		},
		{
			name: "one that is not a catalog",
			path: filepath.Join("testdata", "bad.json"),
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := catalogview.Open(tt.path)

			if tt.err {
				s.Require().Error(err)

				if tt.says != "" {
					s.Require().Contains(err.Error(), tt.says)
				}

				return
			}

			s.Require().NoError(err)

			if tt.device != "" {
				s.Require().Equal(tt.device, got.Device)
				s.Require().Len(got.Blocks, tt.blocks)

				return
			}

			s.Require().NotEmpty(got.Blocks)
			s.Require().NotEmpty(got.Source)
		})
	}
}

// TestList writes out the blocks a filter selects.
func (s *CatalogViewPublicTestSuite) TestList() {
	tests := []struct {
		name   string
		path   string
		filter catalogview.Filter
		// the identifiers the answer must carry, and must not.
		contains []string
		absent   []string
		// nothing came through the filter.
		empty bool
		// what the catalog says about itself. The source is matched loosely,
		// since it carries a version this test has no business pinning.
		total  int
		source string
		err    bool
	}{
		{
			name:     "every block, with a count",
			path:     s.path(),
			contains: []string{"HD2_AmpTestBass", "HD2_DriveTest"},
			total:    2,
		},
		{
			name:     "narrowed to a category",
			path:     s.path(),
			filter:   catalogview.Filter{Category: "amp"},
			contains: []string{"HD2_AmpTestBass"},
			absent:   []string{"HD2_DriveTest"},
			total:    2,
		},
		{
			// This is the filter that makes a bass request draw from bass
			// amps.
			name:     "narrowed to a subcategory",
			path:     s.path(),
			filter:   catalogview.Filter{Subcategory: "bass"},
			contains: []string{"HD2_AmpTestBass"},
			absent:   []string{"HD2_DriveTest"},
		},
		{
			name:     "searched by the name a person would use",
			path:     s.path(),
			filter:   catalogview.Filter{Search: "Test Bass"},
			contains: []string{"HD2_AmpTestBass"},
		},
		{
			name:     "searched by the gear it emulates",
			path:     s.path(),
			filter:   catalogview.Filter{Search: "ampeg"},
			contains: []string{"HD2_AmpTestBass"},
		},
		{
			name:     "searched by identifier, whatever case",
			path:     s.path(),
			filter:   catalogview.Filter{Search: "hd2_amptestbass"},
			contains: []string{"HD2_AmpTestBass"},
		},
		{
			// A device's own name for a model is not what the catalog
			// searches, so this finds nothing rather than everything.
			name:   "searched by a name only the device uses",
			path:   s.path(),
			filter: catalogview.Filter{Search: "SVBeast"},
			empty:  true,
		},
		{
			name:   "a filter nothing matches",
			path:   s.path(),
			filter: catalogview.Filter{Category: "looper"},
			empty:  true,
		},
		{
			// A catalog is only true of the release it came from, so it says
			// which.
			name:   "the release it came from",
			path:   "",
			filter: catalogview.Filter{Search: "klon"},
			source: "HX Edit",
		},
		{
			// Nothing recorded it, which is a different thing from a source
			// nobody recognises. What to say about that is the renderer's.
			name:   "one that cannot name its source",
			path:   s.path(),
			source: "",
		},
		{
			name: "a catalog that will not open",
			path: filepath.Join("testdata", "nope.json"),
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			blocks, err := catalogview.List(tt.path, tt.filter)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			got := make([]string, 0, len(blocks.Matched))
			for _, b := range blocks.Matched {
				got = append(got, string(b.ID))
			}

			for _, want := range tt.contains {
				s.Require().Contains(got, want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(got, gone)
			}

			if tt.empty {
				s.Require().Empty(blocks.Matched)
			}

			if tt.total != 0 {
				s.Require().Equal(tt.total, blocks.Total)
			}

			s.Require().Contains(blocks.Source, tt.source)
		})
	}
}

// TestShow writes out one block and everything it accepts.
func (s *CatalogViewPublicTestSuite) TestShow() {
	tests := []struct {
		name string
		path string
		id   string
		// the parameters the block must accept.
		params []string
		err    bool
	}{
		{
			name:   "a block and its parameters",
			path:   s.path(),
			id:     "HD2_AmpTestBass",
			params: []string{"Drive", "Bright"},
		},
		{
			name: "a block the catalog does not carry",
			path: s.path(),
			id:   "HD2_Nope",
			err:  true,
		},
		{
			name: "a catalog that will not open",
			path: filepath.Join("testdata", "nope.json"),
			id:   "x",
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			block, err := catalogview.Show(tt.path, tt.id)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, string(block.ID))

			for _, want := range tt.params {
				s.Require().Contains(block.Params, want)
			}
		})
	}
}

// TestNotFoundError covers what somebody reads when the block is not there.
func (s *CatalogViewPublicTestSuite) TestNotFoundError() {
	err := &catalogview.NotFoundError{ID: "HD2_Nope", Known: 665}

	s.Require().Contains(err.Error(), "HD2_Nope")
	s.Require().Contains(err.Error(), "665")
	s.Require().Contains(err.Error(), "catalog list")
	s.Require().ErrorIs(err, catalogview.ErrNotFound)
}

// TestDefaultPathIsWhereTheCatalogLives keeps the fallback pointing at the
// generated file rather than wherever it used to be.
func (s *CatalogViewPublicTestSuite) TestDefaultPathIsWhereTheCatalogLives() {
	s.Require().Equal(
		"resources/schemas/hx-stomp.catalog.json", catalogview.DefaultPath)
}

func TestCatalogViewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CatalogViewPublicTestSuite))
}
