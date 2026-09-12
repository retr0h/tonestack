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

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
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

// TestBuild turns a licensed HX Edit installation into a catalog.
//
// Every row builds from the same fixture and asks one thing of the result,
// because the arrange and act are identical and only the question changes.
func (s *BuildTestSuite) TestBuild() {
	tests := []struct {
		name  string
		check func(*catalog.Catalog)
	}{
		{
			name: "the device it was generated for",
			check: func(c *catalog.Catalog) {
				s.Require().Equal("HX Stomp", c.Device)
				s.Require().Equal(2162694, c.DeviceID)
				s.Require().Equal(6, c.SchemaVersion)
			},
		},
		{
			name: "a model listing only another device is left out",
			check: func(c *catalog.Catalog) {
				_, ok := c.Block("HD2_AmpOtherDevice")
				s.Require().False(ok)
			},
		},
		{
			name: "one listing no devices at all is universal",
			check: func(c *catalog.Catalog) {
				_, ok := c.Block("HD2_AmpUniversal")
				s.Require().True(ok)
			},
		},
		{
			name: "the gear map names what a model emulates",
			check: func(c *catalog.Catalog) {
				b, ok := c.Block("HD2_AmpTestBass")
				s.Require().True(ok)
				s.Require().Equal("Ampeg SVT® (normal channel)", b.BasedOn)
				s.Require().Equal("Bass", b.Subcategory)
			},
		},
		{
			name: "a cost Line 6 states is carried and trusted",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpTestBass")
				s.Require().InDelta(26.67, b.DSP.Mono, 1e-9)
				s.Require().InDelta(40.1, b.DSP.Stereo, 1e-9)
				s.Require().Equal(catalog.ProvOfficial, b.DSP.Prov)
				s.Require().True(b.DSP.Prov.Trusted())
			},
		},
		{
			name: "one nobody stated is marked assumed and not trusted",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpUniversal")
				s.Require().Equal(catalog.ProvAssumed, b.DSP.Prov)
				s.Require().False(b.DSP.Prov.Trusted(),
					"a block with no stated cost must not reach a user")
			},
		},
		{
			name: "a float parameter, with its range and default",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpTestBass")
				p := b.Params["Drive"]

				s.Require().Equal(catalog.ParamFloat, p.Type)
				s.Require().InDelta(1.0, p.Max, 1e-9)

				got, ok := p.Default.Float()
				s.Require().True(ok)
				s.Require().InDelta(0.53, got, 1e-9)
			},
		},
		{
			name: "a whole-numbered one",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpTestBass")
				p := b.Params["Taps"]

				s.Require().Equal(catalog.ParamInt, p.Type)
				s.Require().InDelta(3, p.Max, 1e-9)

				got, ok := p.Default.Int()
				s.Require().True(ok)
				s.Require().Equal(int64(1), got)
			},
		},
		{
			name: "a switch",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpTestBass")
				p := b.Params["Bright"]

				s.Require().Equal(catalog.ParamBool, p.Type)

				got, ok := p.Default.Bool()
				s.Require().True(ok)
				s.Require().True(got)
			},
		},
		{
			name: "one chosen from a list",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpTestBass")
				p := b.Params["Topology"]

				s.Require().Equal(catalog.ParamEnum, p.Type)

				_, ok := p.Default.Enum()
				s.Require().True(ok)
			},
		},
		{
			name: "neither a switch nor a list has a range",
			check: func(c *catalog.Catalog) {
				b, _ := c.Block("HD2_AmpTestBass")

				s.Require().Zero(b.Params["Bright"].Max, "false/true is not a range")
				s.Require().Zero(b.Params["Topology"].Max, "empty strings are not a range")
			},
		},
		{
			// Regression: Line 6 lists block attributes alongside knobs.
			// Carrying @enabled or @bypassvolume as parameters puts a value
			// we chose where the device owns one, and writes it into every
			// generated preset.
			name: "an @-prefixed entry is an attribute, not a knob",
			check: func(c *catalog.Catalog) {
				b, ok := c.Block("HD2_AmpTestBass")
				s.Require().True(ok)

				for key := range b.Params {
					s.Require().NotContains(key, "@",
						"attribute %q became a parameter", key)
				}
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := Build(s.opts())

			s.Require().NoError(err)
			tt.check(got)
		})
	}
}

// TestBuildWithoutAGearMap covers the half of the sources that is optional.
//
// The catalog is usable without it: the models supply every value, and the map
// only says what real gear each one emulates. Naming no map is that choice;
// naming one that is not there is a mistake, and covered beside the other
// failures.
func (s *BuildTestSuite) TestBuildWithoutAGearMap() {
	o := s.opts()
	o.GearMapPath = ""

	got, err := Build(o)
	s.Require().NoError(err)

	b, ok := got.Block("HD2_AmpTestBass")
	s.Require().True(ok)
	s.Require().Empty(b.BasedOn, "usable, just unable to name gear")
}

// TestBuildReportsWhatItCannotRead covers the sources going missing or
// arriving malformed.
func (s *BuildTestSuite) TestBuildReportsWhatItCannotRead() {
	unreadable := s.T().TempDir()
	path := filepath.Join(unreadable, "amp.models")
	s.Require().NoError(os.WriteFile(path, []byte("[]"), 0o600))
	s.Require().NoError(os.Chmod(path, 0o000))

	tests := []struct {
		name      string
		resources string
		gearMap   string
		is        error
		says      string
	}{
		{
			name:      "a resources directory that is not there",
			resources: filepath.Join("testdata", "does-not-exist"),
			is:        ErrNoResources,
		},
		{
			// A directory holding only models that name another device. The
			// default fixture also holds one listing no devices at all,
			// which is universal and would always be included.
			name:      "a directory where no model supports this device",
			resources: filepath.Join("testdata", "otheronly"),
			is:        ErrNoResources,
		},
		{
			// filepath.Glob rejects an unterminated character class.
			name:      "a directory name no pattern can match",
			resources: filepath.Join("testdata", "["),
		},
		{
			name:      "model definitions that will not decode",
			resources: filepath.Join("testdata", "malformed"),
			says:      "decoding",
		},
		{
			name:      "a model file that cannot be read",
			resources: unreadable,
			says:      "reading",
		},
		{
			name:    "a gear map that will not decode",
			gearMap: filepath.Join("testdata", "badmap.json"),
			says:    "gear map",
		},
		{
			name:    "a gear map that is a directory",
			gearMap: "testdata",
			says:    "gear map",
		},
		{
			// The file is gitignored, so a fresh clone has none. Swallowing
			// this generated a catalog where no recipe could resolve any
			// gear, reported only as "0 mapped to real gear".
			name:    "a gear map somebody named and does not have",
			gearMap: filepath.Join("testdata", "no-such-map.json"),
			says:    "gear map",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			o := s.opts()

			if tt.resources != "" {
				o.ResourcesDir = tt.resources
			}

			if tt.gearMap != "" {
				o.GearMapPath = tt.gearMap
			}

			_, err := Build(o)

			s.Require().Error(err)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}

			if tt.says != "" {
				s.Require().Contains(err.Error(), tt.says)
			}
		})
	}
}

func TestBuildTestSuite(t *testing.T) {
	suite.Run(t, new(BuildTestSuite))
}
