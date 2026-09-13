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
package catalogen

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

type RefreshTestSuite struct {
	suite.Suite
}

func (s *RefreshTestSuite) opts(out string) Options {
	return Options{
		ResourcesDir: "testdata",
		GearMapPath:  filepath.Join("testdata", "gear-map.json"),
		DeviceID:     2162694,
		DeviceName:   "HX Stomp",
		OutputPath:   out,
	}
}

// written reads back a catalog this suite generated.
func (s *RefreshTestSuite) written(path string) *catalog.Catalog {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	zr, err := gzip.NewReader(f)
	s.Require().NoError(err)

	c, err := catalog.Load(zr)
	s.Require().NoError(err)

	return c
}

// TestRefresh covers building a catalog out of somebody's HX Edit
// installation, or not, when the machine has none.
func (s *RefreshTestSuite) TestRefresh() {
	tests := []struct {
		name      string
		resources string
		gearMap   string
		out       string
		// what the answer must say.
		skipped    string
		wantSchema int
		errText    string
	}{
		{
			name: "a catalog, and an answer about what went into it",
			// Nobody said which version to write, so it takes the current one.
			wantSchema: 6,
		},
		{
			// CI and most contributors: nothing to build from, and nothing
			// broken either.
			name:      "a machine without HX Edit",
			resources: filepath.Join("testdata", "does-not-exist"),
			skipped:   "no HX Edit",
		},
		{
			name:    "a machine without the gear map",
			gearMap: filepath.Join("testdata", "no-gear-map.json"),
			skipped: "run just gear-map",
		},
		{
			// Every input is there and one of them is wrong. That is a
			// failure to report, not a machine to skip.
			name:    "a gear map that will not parse",
			gearMap: filepath.Join("testdata", "badmap.json"),
			errText: "gear map",
		},
		{
			name:    "nowhere to write it",
			out:     filepath.Join("no", "such", "dir.json"),
			errText: "writing",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "catalog.json")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			o := s.opts(out)
			if tt.resources != "" {
				o.ResourcesDir = tt.resources
			}

			if tt.gearMap != "" {
				o.GearMapPath = tt.gearMap
			}

			got, err := Refresh(o)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			if tt.skipped != "" {
				s.Require().Contains(got.Skipped, tt.skipped)
				s.Require().False(got.Changed)
				s.Require().NoFileExists(out)

				return
			}

			s.Require().True(got.Changed)
			s.Require().Equal(tt.wantSchema, s.written(out).SchemaVersion)
			s.Require().Equal(out, got.Path)
			s.Require().NotZero(got.Blocks)
			s.Require().NotZero(got.Named)
		})
	}
}

// TestRefreshLeavesAnUnchangedCatalogAlone covers running it again.
//
// go generate runs on every `just ready`. A catalog that has not changed is
// not written, so nothing shows up as a change nobody made.
func (s *RefreshTestSuite) TestRefreshLeavesAnUnchangedCatalogAlone() {
	out := filepath.Join(s.T().TempDir(), "catalog.json")

	first, err := Refresh(s.opts(out))
	s.Require().NoError(err)
	s.Require().True(first.Changed)

	was, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	second, err := Refresh(s.opts(out))
	s.Require().NoError(err)
	s.Require().False(second.Changed)
	s.Require().Equal(first.Blocks, second.Blocks)

	now, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)
	s.Require().Equal(was, now)
}

func TestRefreshTestSuite(t *testing.T) {
	suite.Run(t, new(RefreshTestSuite))
}
