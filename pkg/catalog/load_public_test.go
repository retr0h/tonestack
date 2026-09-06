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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type LoadPublicTestSuite struct {
	suite.Suite
}

func (s *LoadPublicTestSuite) loadMinimal() *catalog.Catalog {
	s.T().Helper()

	f, err := os.Open("testdata/minimal.json")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	c, err := catalog.Load(f)
	s.Require().NoError(err)

	return c
}

func (s *LoadPublicTestSuite) TestLoadReadsDeviceIdentity() {
	c := s.loadMinimal()

	s.Require().Equal("HX Stomp", c.Device)
	s.Require().Equal(2162689, c.DeviceID)
	s.Require().Equal(6, c.SchemaVersion)
	s.Require().Equal(30, c.ModelData)
}

func (s *LoadPublicTestSuite) TestLoadReadsBlockAndParams() {
	c := s.loadMinimal()

	b, ok := c.Block("HD2_AmpTest")
	s.Require().True(ok)
	s.Require().Equal("Test Amp", b.Name)
	s.Require().Equal(catalog.CategoryAmp, b.Category)
	s.Require().False(b.Stereo)
	s.Require().Equal(catalog.ProvObserved, b.Prov)
	s.Require().InDelta(0.30, b.DSP.Mono, 1e-9)
	s.Require().Equal(catalog.ProvMeasured, b.DSP.Prov)

	gain, ok := b.Params["Gain"]
	s.Require().True(ok)
	s.Require().Equal(catalog.ParamFloat, gain.Type)
	s.Require().InDelta(1.0, gain.Max, 1e-9)

	mode, ok := b.Params["Mode"]
	s.Require().True(ok)
	s.Require().Equal(catalog.ParamEnum, mode.Type)
	s.Require().Equal([]string{"Normal", "Bright"}, mode.Enum)
}

func (s *LoadPublicTestSuite) TestBlockReportsFalseForUnknownModel() {
	c := s.loadMinimal()

	_, ok := c.Block("HD2_Nope")
	s.Require().False(ok)
}

func (s *LoadPublicTestSuite) TestLoadRejectsMalformedJSON() {
	_, err := catalog.Load(strings.NewReader("{not json"))

	s.Require().Error(err)
}

func (s *LoadPublicTestSuite) TestLoadRejectsAReaderThatFails() {
	_, err := catalog.Load(&failingReader{})

	s.Require().Error(err)
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, os.ErrClosed }

func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
