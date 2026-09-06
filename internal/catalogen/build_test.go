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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type BuildTestSuite struct {
	suite.Suite
}

func (s *BuildTestSuite) opts() Options {
	return Options{
		ResourcesDir:  "testdata",
		GearMapPath:   filepath.Join("testdata", "gear-map.json"),
		DeviceID:      2162694,
		DeviceName:    "HX Stomp",
		SchemaVersion: 6,
	}
}

func (s *BuildTestSuite) TestBuildRecordsDeviceIdentity() {
	c, err := Build(s.opts())

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", c.Device)
	s.Require().Equal(2162694, c.DeviceID)
	s.Require().Equal(6, c.SchemaVersion)
}

func (s *BuildTestSuite) TestBuildExcludesModelsTheDeviceLacks() {
	c, err := Build(s.opts())

	s.Require().NoError(err)
	_, ok := c.Block("HD2_AmpOtherDevice")
	s.Require().False(ok, "a model listing only another device must be excluded")
}

func (s *BuildTestSuite) TestBuildTreatsAnEmptyDeviceListAsUniversal() {
	c, err := Build(s.opts())

	s.Require().NoError(err)
	_, ok := c.Block("HD2_AmpUniversal")
	s.Require().True(ok)
}

func (s *BuildTestSuite) TestBuildJoinsTheGearMap() {
	c, err := Build(s.opts())
	s.Require().NoError(err)

	b, ok := c.Block("HD2_AmpTestBass")
	s.Require().True(ok)
	s.Require().Equal("Ampeg SVT® (normal channel)", b.BasedOn)
	s.Require().Equal("Bass", b.Subcategory)
}

func (s *BuildTestSuite) TestBuildCarriesOfficialDSPCost() {
	c, _ := Build(s.opts())
	b, _ := c.Block("HD2_AmpTestBass")

	s.Require().InDelta(26.67, b.DSP.Mono, 1e-9)
	s.Require().InDelta(40.1, b.DSP.Stereo, 1e-9)
	s.Require().Equal(catalog.ProvOfficial, b.DSP.Prov)
	s.Require().True(b.DSP.Prov.Trusted())
}

func (s *BuildTestSuite) TestBuildMarksAMissingDSPCostAssumed() {
	c, _ := Build(s.opts())
	b, _ := c.Block("HD2_AmpUniversal")

	s.Require().Equal(catalog.ProvAssumed, b.DSP.Prov)
	s.Require().False(b.DSP.Prov.Trusted(),
		"a block with no stated cost must not reach a user")
}

func (s *BuildTestSuite) TestBuildDecodesEveryParameterKind() {
	c, _ := Build(s.opts())
	b, _ := c.Block("HD2_AmpTestBass")

	f := b.Params["Drive"]
	s.Require().Equal(catalog.ParamFloat, f.Type)
	s.Require().InDelta(1.0, f.Max, 1e-9)
	fv, ok := f.Default.Float()
	s.Require().True(ok)
	s.Require().InDelta(0.53, fv, 1e-9)

	i := b.Params["Taps"]
	s.Require().Equal(catalog.ParamInt, i.Type)
	s.Require().InDelta(3, i.Max, 1e-9)
	iv, ok := i.Default.Int()
	s.Require().True(ok)
	s.Require().Equal(int64(1), iv)

	bl := b.Params["Bright"]
	s.Require().Equal(catalog.ParamBool, bl.Type)
	bv, ok := bl.Default.Bool()
	s.Require().True(ok)
	s.Require().True(bv)

	e := b.Params["Topology"]
	s.Require().Equal(catalog.ParamEnum, e.Type)
	_, ok = e.Default.Enum()
	s.Require().True(ok)
}

func (s *BuildTestSuite) TestBuildLeavesBoolAndStringWithoutARange() {
	c, _ := Build(s.opts())
	b, _ := c.Block("HD2_AmpTestBass")

	s.Require().Zero(b.Params["Bright"].Max, "false/true is not a range")
	s.Require().Zero(b.Params["Topology"].Max, "empty strings are not a range")
}

func (s *BuildTestSuite) TestBuildReportsAMissingResourcesDir() {
	o := s.opts()
	o.ResourcesDir = "testdata/does-not-exist"

	_, err := Build(o)

	s.Require().ErrorIs(err, ErrNoResources)
}

func (s *BuildTestSuite) TestBuildReportsADeviceNoModelSupports() {
	// A directory holding only models that name another device. The default
	// fixture also holds a model listing no devices at all, which is treated
	// as universal and would always be included.
	o := s.opts()
	o.ResourcesDir = filepath.Join("testdata", "otheronly")

	_, err := Build(o)

	s.Require().ErrorIs(err, ErrNoResources)
}

func (s *BuildTestSuite) TestBuildToleratesAMissingGearMap() {
	o := s.opts()
	o.GearMapPath = "testdata/no-such-map.json"

	c, err := Build(o)

	s.Require().NoError(err)
	b, _ := c.Block("HD2_AmpTestBass")
	s.Require().Empty(b.BasedOn, "usable without the map, just unable to name gear")
}

func (s *BuildTestSuite) TestBuildAcceptsAnEmptyGearMapPath() {
	o := s.opts()
	o.GearMapPath = ""

	_, err := Build(o)

	s.Require().NoError(err)
}

func TestBuildTestSuite(t *testing.T) {
	suite.Run(t, new(BuildTestSuite))
}

func (s *BuildTestSuite) TestBuildReportsMalformedModelDefinitions() {
	o := s.opts()
	o.ResourcesDir = filepath.Join("testdata", "malformed")

	_, err := Build(o)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "decoding")
}

func (s *BuildTestSuite) TestBuildReportsAMalformedGearMap() {
	o := s.opts()
	o.GearMapPath = filepath.Join("testdata", "badmap.json")

	_, err := Build(o)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "gear map")
}

func (s *BuildTestSuite) TestBuildReportsABadResourcesPattern() {
	// filepath.Glob rejects an unterminated character class.
	o := s.opts()
	o.ResourcesDir = "testdata/["

	_, err := Build(o)

	s.Require().Error(err)
}

func (s *BuildTestSuite) TestBuildReportsAnUnreadableModelFile() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "amp.models")
	s.Require().NoError(os.WriteFile(path, []byte("[]"), 0o600))
	s.Require().NoError(os.Chmod(path, 0o000))

	o := s.opts()
	o.ResourcesDir = dir

	_, err := Build(o)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reading")
}

func (s *BuildTestSuite) TestBuildReportsAnUnreadableGearMap() {
	// A directory is not ErrNotExist, so it is a real read failure rather
	// than an absent map.
	o := s.opts()
	o.GearMapPath = "testdata"

	_, err := Build(o)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "gear map")
}
