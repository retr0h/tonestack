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

package rig_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/rig"
)

type ValidateParamsPublicTestSuite struct {
	suite.Suite
}

func (*ValidateParamsPublicTestSuite) specWith(
	params map[string]catalog.ParamValue,
) rig.Spec {
	return rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", Params: params}},
	}
}

func (s *ValidateParamsPublicTestSuite) TestAcceptsValuesInRange() {
	spec := s.specWith(map[string]catalog.ParamValue{
		"Gain": catalog.Float(0.5),
		"Mode": catalog.Enum("Bright"),
	})

	s.Require().NoError(rig.ValidateParams(newCatalog(testAmp()), spec))
}

func (s *ValidateParamsPublicTestSuite) TestAcceptsValuesOnTheBoundary() {
	for _, v := range []float64{0.0, 1.0} {
		spec := s.specWith(
			map[string]catalog.ParamValue{"Gain": catalog.Float(v)},
		)
		s.Require().NoError(rig.ValidateParams(newCatalog(testAmp()), spec))
	}
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnUnknownParameter() {
	spec := s.specWith(
		map[string]catalog.ParamValue{"Nope": catalog.Float(0.5)},
	)

	err := rig.ValidateParams(newCatalog(testAmp()), spec)

	s.Require().ErrorIs(err, catalog.ErrBadParam)

	var target *catalog.BadParamError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal("Nope", target.Key)
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAWrongType() {
	spec := s.specWith(
		map[string]catalog.ParamValue{"Gain": catalog.Enum("loud")},
	)

	err := rig.ValidateParams(newCatalog(testAmp()), spec)

	s.Require().ErrorIs(err, catalog.ErrBadParam)
	s.Require().Contains(err.Error(), "expected float")
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAFloatOutOfRange() {
	for _, v := range []float64{-0.01, 1.01} {
		spec := s.specWith(
			map[string]catalog.ParamValue{"Gain": catalog.Float(v)},
		)

		err := rig.ValidateParams(newCatalog(testAmp()), spec)

		s.Require().ErrorIs(err, catalog.ErrBadParam)
		s.Require().Contains(err.Error(), "out of range")
	}
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnIntOutOfRange() {
	blk := testAmp()
	blk.Params["Taps"] = catalog.Param{
		Key: "Taps", Type: catalog.ParamInt, Min: 1, Max: 4, Default: catalog.Int(1),
	}

	spec := s.specWith(map[string]catalog.ParamValue{"Taps": catalog.Int(9)})

	err := rig.ValidateParams(newCatalog(blk), spec)

	s.Require().ErrorIs(err, catalog.ErrBadParam)
}

func (s *ValidateParamsPublicTestSuite) TestAcceptsABoolWithoutRangeChecking() {
	blk := testAmp()
	blk.Params["Bright"] = catalog.Param{
		Key: "Bright", Type: catalog.ParamBool, Default: catalog.Bool(false),
	}

	spec := s.specWith(
		map[string]catalog.ParamValue{"Bright": catalog.Bool(true)},
	)

	s.Require().NoError(rig.ValidateParams(newCatalog(blk), spec))
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnEnumMemberNotDeclared() {
	spec := s.specWith(
		map[string]catalog.ParamValue{"Mode": catalog.Enum("Sparkle")},
	)

	err := rig.ValidateParams(newCatalog(testAmp()), spec)

	s.Require().ErrorIs(err, catalog.ErrBadParam)
	s.Require().Contains(err.Error(), "Sparkle")
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAKindTheCatalogInvented() {
	blk := testAmp()
	blk.Params["Weird"] = catalog.Param{Key: "Weird", Type: catalog.ParamType("wat")}

	spec := s.specWith(map[string]catalog.ParamValue{"Weird": catalog.Float(1)})

	err := rig.ValidateParams(newCatalog(blk), spec)

	s.Require().ErrorIs(err, catalog.ErrBadParam)
	s.Require().Contains(err.Error(), "unknown kind")
}

func (s *ValidateParamsPublicTestSuite) TestRejectsEachKindMismatch() {
	blk := testAmp()
	blk.Params["Taps"] = catalog.Param{Key: "Taps", Type: catalog.ParamInt, Min: 0, Max: 9}
	blk.Params["Bright"] = catalog.Param{Key: "Bright", Type: catalog.ParamBool}

	tests := []struct {
		name string
		key  string
		val  catalog.ParamValue
	}{
		{"float declared, enum given", "Gain", catalog.Enum("loud")},
		{"int declared, float given", "Taps", catalog.Float(1)},
		{"bool declared, int given", "Bright", catalog.Int(1)},
		{"enum declared, bool given", "Mode", catalog.Bool(true)},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			spec := s.specWith(map[string]catalog.ParamValue{tc.key: tc.val})

			err := rig.ValidateParams(newCatalog(blk), spec)

			s.Require().ErrorIs(err, catalog.ErrBadParam)
			s.Require().Contains(err.Error(), "expected")
		})
	}
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnUnknownModel() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_Nope"}}}

	err := rig.ValidateParams(newCatalog(testAmp()), spec)

	s.Require().ErrorIs(err, rig.ErrUnknownBlock)
}

func (s *ValidateParamsPublicTestSuite) TestReportsTheFirstBadParameterInSortedOrder() {
	spec := s.specWith(map[string]catalog.ParamValue{
		"Zebra": catalog.Float(0.5),
		"Alpha": catalog.Float(0.5),
	})

	for range 20 {
		err := rig.ValidateParams(newCatalog(testAmp()), spec)

		var target *catalog.BadParamError
		s.Require().True(errors.As(err, &target))
		s.Require().
			Equal("Alpha", target.Key, "must be deterministic across runs")
	}
}

func TestValidateParamsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateParamsPublicTestSuite))
}
