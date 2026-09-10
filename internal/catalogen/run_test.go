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

type RunTestSuite struct {
	suite.Suite
}

func (s *RunTestSuite) opts(out string) Options {
	return Options{
		ResourcesDir: "testdata",
		GearMapPath:  filepath.Join("testdata", "gear-map.json"),
		DeviceID:     2162694,
		DeviceName:   "HX Stomp",
		OutputPath:   out,
	}
}

// written reads back a catalog this suite generated.
//
// The file is gzipped, because it is embedded in the binary and a catalog
// compresses to a twentieth of its size.
func (s *RunTestSuite) written(path string) *catalog.Catalog {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	zr, err := gzip.NewReader(f)
	s.Require().NoError(err)

	c, err := catalog.Load(zr)
	s.Require().NoError(err)

	return c
}

// TestRun builds a catalog out of somebody's HX Edit installation.
func (s *RunTestSuite) TestRun() {
	tests := []struct {
		name      string
		resources string
		out       string
		// what the answer must say about what it built.
		device     string
		named      bool
		wantSchema int
		err        error
		errText    string
	}{
		{
			name:   "a catalog, and an answer about what went into it",
			device: "HX Stomp",
			named:  true,
			// Nobody said which version to write, so it takes the current one.
			wantSchema: 6,
		},
		{
			name:      "resources that are not there",
			resources: filepath.Join("testdata", "does-not-exist"),
			err:       ErrNoResources,
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

			built, err := Run(o)

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
			s.Require().FileExists(out)
			s.Require().Equal(tt.wantSchema, s.written(out).SchemaVersion)
			s.Require().Equal(out, built.Path)
			s.Require().Equal(tt.device, built.Device)
			s.Require().NotZero(built.Blocks)

			if tt.named {
				s.Require().NotZero(built.Named)
			}
		})
	}
}

func TestRunTestSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}
