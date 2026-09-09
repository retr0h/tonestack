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
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type BlockTestSuite struct {
	suite.Suite
}

// TestCategory maps every family Line 6 name to one of ours.
func (s *BlockTestSuite) TestCategory() {
	tests := map[string]catalog.Category{
		"amp":              catalog.CategoryAmp,
		"preamp":           catalog.CategoryAmp,
		"cab":              catalog.CategoryCab,
		"cabmicirs":        catalog.CategoryCab,
		"cabmicirswithpan": catalog.CategoryCab,
		"distortion":       catalog.CategoryDrive,
		"compressor":       catalog.CategoryComp,
		"gate":             catalog.CategoryGate,
		"delay":            catalog.CategoryDelay,
		"reverb":           catalog.CategoryReverb,
		"eq":               catalog.CategoryEQ,
		"modulation":       catalog.CategoryMod,
		"wah":              catalog.CategoryWah,
		"pitch-synth":      catalog.CategoryPitch,
		"filter":           catalog.CategoryFilter,
		"volumepan":        catalog.CategoryUtility,
		"sendreturn":       catalog.CategoryUtility,
		"io":               catalog.CategoryUtility,
		"fixed":            catalog.CategoryUtility,
		"looper":           catalog.CategoryUtility,
		"anything-else":    catalog.CategoryOther,
	}

	for family, want := range tests {
		s.Run(family, func() {
			s.Require().Equal(want, category(family))
		})
	}
}

// TestNumber reads a value that is meant to be one.
func (s *BlockTestSuite) TestNumber() {
	tests := []struct {
		name string
		raw  json.RawMessage
	}{
		{name: "something that is not a number", raw: json.RawMessage(`"not a number"`)},
		{name: "nothing at all", raw: json.RawMessage(``)},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Zero(number(tt.raw))
		})
	}
}

// TestDefaultValue reads the value Line 6 state a knob starts at.
func (s *BlockTestSuite) TestDefaultValue() {
	tests := []struct {
		name string
		kind int
		raw  json.RawMessage
		// the integer it must come back as, when it is one.
		want int64
	}{
		{
			// Line 6 records four integer defaults as floats. Reading those
			// as ints fails, and the value is recovered through the float
			// path.
			name: "an integer written as a float",
			kind: wireInt,
			raw:  json.RawMessage("2.0"),
			want: 2,
		},
		{
			name: "a switch that will not decode",
			kind: wireBool,
			raw:  json.RawMessage(`{}`),
		},
		{
			name: "a string that will not decode",
			kind: wireString,
			raw:  json.RawMessage(`{}`),
		},
		{
			name: "a float that will not decode",
			kind: wireFloat,
			raw:  json.RawMessage(`{}`),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := defaultValue(wireParam{ValueType: tt.kind, Default: tt.raw})

			s.Require().NotEqual(catalog.ParamType(""), got.Type(),
				"a value that will not decode still has a kind")

			if tt.want == 0 {
				return
			}

			v, ok := got.Int()
			s.Require().True(ok)
			s.Require().Equal(tt.want, v)
		})
	}
}

// TestBuild carries a family through to the catalog it writes.
func (s *BlockTestSuite) TestBuild() {
	c, err := Build(Options{
		ResourcesDir: filepath.Join("testdata", "families"),
		DeviceID:     2162694,
		DeviceName:   "HX Stomp",
	})

	s.Require().NoError(err)
	b, ok := c.Block("HD2_cabOne")
	s.Require().True(ok)
	s.Require().Equal(catalog.CategoryCab, b.Category)
}

func TestBlockTestSuite(t *testing.T) {
	suite.Run(t, new(BlockTestSuite))
}
