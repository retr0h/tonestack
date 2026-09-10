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

package catalog_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

type OpenPublicTestSuite struct {
	suite.Suite
}

// TestOpen covers reading a catalog from wherever it is.
func (s *OpenPublicTestSuite) TestOpen() {
	tests := []struct {
		name string
		path string
		ok   bool
	}{
		{
			// No path means the catalog that ships in the binary, which is
			// the case for anyone who has not generated their own.
			name: "the catalog in the binary",
			path: "",
			ok:   true,
		},
		{
			name: "one on disk",
			path: filepath.Join("testdata", "minimal.json"),
			ok:   true,
		},
		{name: "one that is not there", path: filepath.Join("testdata", "nope.json")},
		{
			// A file that opens and is not a catalog fails at the decode
			// rather than at the open, and both have to be reported.
			name: "one that is not a catalog",
			path: filepath.Join("testdata", "notjson.json"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := catalog.Open(tt.path)

			if !tt.ok {
				s.Require().Error(err)
				s.Require().Nil(got)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(got.Blocks)
		})
	}
}

// TestFilesOpen covers reading one through the value a caller is given when
// it says nothing about where catalogs come from.
func (s *OpenPublicTestSuite) TestFilesOpen() {
	tests := []struct {
		name string
		path string
		ok   bool
	}{
		{name: "the catalog in the binary", path: "", ok: true},
		{
			name: "one on disk",
			path: filepath.Join("testdata", "minimal.json"),
			ok:   true,
		},
		{name: "one that is not there", path: filepath.Join("testdata", "nope.json")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := catalog.Files{}.Open(tt.path)

			if !tt.ok {
				s.Require().Error(err)
				s.Require().Nil(got)

				return
			}

			s.Require().NoError(err)

			// The same catalog either way; the value is a seam, not a
			// second way of reading one.
			want, err := catalog.Open(tt.path)
			s.Require().NoError(err)
			s.Require().Equal(want, got)
		})
	}
}

// TestDefaultPathIsWhereTheCatalogLives pins where a generated catalog goes.
func (s *OpenPublicTestSuite) TestDefaultPathIsWhereTheCatalogLives() {
	s.Require().Equal(
		"resources/schemas/hx-stomp.catalog.json", catalog.DefaultPath)
}

func TestOpenPublicTestSuite(t *testing.T) {
	suite.Run(t, new(OpenPublicTestSuite))
}
