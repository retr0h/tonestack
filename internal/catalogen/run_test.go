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
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
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

func (s *RunTestSuite) TestRunWritesACatalogAndReportsIt() {
	out := filepath.Join(s.T().TempDir(), "catalog.json")

	var log bytes.Buffer
	s.Require().NoError(Run(&log, s.opts(out)))

	s.Require().FileExists(out)
	s.Require().Contains(log.String(), "blocks for HX Stomp")
	s.Require().Contains(log.String(), "mapped to real gear")
}

func (s *RunTestSuite) TestRunDefaultsTheSchemaVersion() {
	out := filepath.Join(s.T().TempDir(), "catalog.json")

	o := s.opts(out)
	o.SchemaVersion = 0
	s.Require().NoError(Run(&bytes.Buffer{}, o))

	s.Require().Equal(6, s.written(out).SchemaVersion)
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

func (s *RunTestSuite) TestRunReportsAFailedBuild() {
	o := s.opts(filepath.Join(s.T().TempDir(), "catalog.json"))
	o.ResourcesDir = "testdata/does-not-exist"

	s.Require().ErrorIs(Run(&bytes.Buffer{}, o), ErrNoResources)
}

func (s *RunTestSuite) TestRunReportsAnUnwritableDestination() {
	err := Run(&bytes.Buffer{}, s.opts(filepath.Join(s.T().TempDir(), "no", "such", "dir.json")))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing")
}

func (s *RunTestSuite) TestRunReportsAFailingWriter() {
	out := filepath.Join(s.T().TempDir(), "catalog.json")

	err := Run(&failingWriter{}, s.opts(out))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reporting")
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestRunTestSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}
