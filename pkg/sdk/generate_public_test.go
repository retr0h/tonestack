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

package sdk_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// GeneratePublicTestSuite covers the two operations that rebuild what this
// library embeds.
//
// A maintainer runs these after Line 6 ship a new release, against a licensed
// HX Edit installation and a corpus nobody redistributes. The fixtures here
// stand in for both.
type GeneratePublicTestSuite struct {
	suite.Suite
}

// under names a fixture belonging to whichever generator owns it.
func under(pkg string, parts ...string) string {
	return filepath.Join(append(
		[]string{"internal", pkg, "testdata"}, parts...)...)
}

// TestGenerateCatalog covers building a device catalog.
func (s *GeneratePublicTestSuite) TestGenerateCatalog() {
	tests := []struct {
		name      string
		resources string
		err       bool
	}{
		{name: "a catalog and what went into it", resources: under("catalogen")},
		{
			name:      "resources that are not there",
			resources: under("catalogen", "does-not-exist"),
			err:       true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "catalog.json.gz")

			got, err := sdk.New().GenerateCatalog(sdk.Catalog{
				ResourcesDir: tt.resources,
				GearMapPath:  under("catalogen", "gear-map.json"),
				DeviceName:   "HX Stomp",
				DeviceID:     2162694,
				OutputPath:   out,
			})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().Equal("HX Stomp", got.Device)
			s.Require().NotZero(got.Blocks)
			s.Require().FileExists(out)
		})
	}
}

// TestMeasureCorpus covers measuring what real presets say.
func (s *GeneratePublicTestSuite) TestMeasureCorpus() {
	tests := []struct {
		name   string
		corpus string
		err    bool
	}{
		{
			name:   "statistics, and the measurements beside them",
			corpus: under("corpusgen", "corpus"),
		},
		{
			name:   "a corpus that is not there",
			corpus: under("corpusgen", "nope"),
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "stats.json.gz")

			got, err := sdk.New().MeasureCorpus(sdk.Measure{
				CorpusDir:   tt.corpus,
				CatalogPath: under("corpusgen", "catalog.json"),
				OutputPath:  out,
			})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().NotNil(got.Stats)
			s.Require().NotZero(got.Stats.Presets)
			s.Require().FileExists(out)
		})
	}
}

func TestGeneratePublicTestSuite(t *testing.T) {
	suite.Run(t, new(GeneratePublicTestSuite))
}
