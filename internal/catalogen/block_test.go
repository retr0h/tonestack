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

func (s *BlockTestSuite) TestCategoryMapsEveryFamily() {
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

func (s *BlockTestSuite) TestNumberReportsZeroForANonNumber() {
	s.Require().Zero(number(json.RawMessage(`"not a number"`)))
	s.Require().Zero(number(json.RawMessage(``)))
}

func (s *BlockTestSuite) TestDefaultValueFallsBackWhenAnIntIsWrittenAsAFloat() {
	// Line 6 records four integer defaults as floats. Reading those as ints
	// fails, and the value is recovered through the float path.
	got := defaultValue(wireParam{ValueType: wireInt, Default: json.RawMessage("2.0")})

	v, ok := got.Int()
	s.Require().True(ok)
	s.Require().Equal(int64(2), v)
}

func (s *BlockTestSuite) TestDefaultValueToleratesAnUndecodableDefault() {
	for _, vt := range []int{wireBool, wireString, wireFloat} {
		got := defaultValue(wireParam{ValueType: vt, Default: json.RawMessage(`{}`)})
		s.Require().NotEqual(catalog.ParamType(""), got.Type())
	}
}

func (s *BlockTestSuite) TestBuildMapsEveryFamilyToACategory() {
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
