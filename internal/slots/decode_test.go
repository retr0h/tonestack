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

package slots

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// DecodeTestSuite covers turning a device's answer into a chain.
//
// The device names nothing: a block is a number into its own model table and
// its parameters are a bare array. Everything here is about putting the names
// back, and about saying so plainly when they cannot be.
type DecodeTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *DecodeTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

func (s *DecodeTestSuite) TestNamesWhatTheDeviceNumbered() {
	got, err := chainOf("BAS:SVT Nrm", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: 5, Values: []any{0.27, 0.65}, Enabled: true},
		{Model: 74, Values: []any{1.0}, Enabled: false},
	}}, s.cat)

	s.Require().NoError(err)
	s.Require().Equal("BAS:SVT Nrm", got.Name)
	s.Require().Len(got.Blocks, 2)

	s.Require().Equal(catalog.ModelID("HD2_AmpSVBeastNrm"), got.Blocks[0].Model)
	s.Require().Equal(catalog.Float(0.27), got.Blocks[0].Params["Drive"])
	s.Require().Equal(catalog.Float(0.65), got.Blocks[0].Params["Bass"])
	s.Require().True(got.Blocks[0].Enabled)

	s.Require().Equal(catalog.ModelID("HD2_Cab8x10SVBeast"), got.Blocks[1].Model)
	s.Require().False(got.Blocks[1].Enabled, "a bypassed block is still a block")
}

func (s *DecodeTestSuite) TestJoinsAMonoInstanceToItsModel() {
	// A device names a mono and a stereo instance separately; the catalog
	// names the model once, the way Line 6's own model files do.
	got, err := chainOf("x", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: 120, Enabled: true},
	}}, s.cat)

	s.Require().NoError(err)
	s.Require().Equal(
		catalog.ModelID("HD2_CompressorLAStudioComp"), got.Blocks[0].Model)
}

func (s *DecodeTestSuite) TestKeepsAModelThisDeviceDoesNotHave() {
	// Symbols cover every Helix; a Stomp has no second effects loop. Keeping
	// the device's own name means the rig still rebuilds it exactly.
	sym, ok := s.cat.Symbol(indexOf(s.cat, "HD2_FXLoopMono3"))
	s.Require().True(ok)
	s.Require().Equal(catalog.ModelID("HD2_FXLoopMono3"), sym.ID)

	got, err := chainOf("x", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: indexOf(s.cat, "HD2_FXLoopMono3")},
	}}, s.cat)

	s.Require().NoError(err)
	s.Require().Equal(catalog.ModelID("HD2_FXLoopMono3"), got.Blocks[0].Model)
}

func (s *DecodeTestSuite) TestReportsAModelTheTableDoesNotReach() {
	_, err := chainOf("x", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: 99999},
	}}, s.cat)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "different release")
}

func (s *DecodeTestSuite) TestReportsACatalogWithNoTable() {
	_, err := chainOf("x", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: 5},
	}}, &catalog.Catalog{})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "catalog generate")
}

func (s *DecodeTestSuite) TestKeepsEachValueInTheShapeItArrivedIn() {
	// A device mixes numbers, switches and enumerated positions in one array,
	// and narrowing them all to numbers turns every switch off. The wire
	// layer has already reduced them to those three kinds.
	got, err := chainOf("x", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: 5, Values: []any{0.27, true, int64(2), "unreadable"}},
	}}, s.cat)

	s.Require().NoError(err)

	p := got.Blocks[0].Params
	s.Require().Equal(catalog.Float(0.27), p["Drive"])
	s.Require().Equal(catalog.Bool(true), p["Bass"])
	s.Require().Equal(catalog.Int(2), p["Mid"])
	s.Require().NotContains(p, "MidFreq", "a value of no known kind is left out")
}

func (s *DecodeTestSuite) TestATruncatedRunOfValues() {
	// A device says less than the table names when it has nothing to say.
	got, err := chainOf("x", wire.DevicePreset{Blocks: []wire.DeviceBlock{
		{Model: 5, Values: []any{0.27}},
	}}, s.cat)

	s.Require().NoError(err)
	s.Require().Len(got.Blocks[0].Params, 1)
}

// indexOf finds where a model sits in the device's own table.
func indexOf(cat *catalog.Catalog, id catalog.ModelID) int {
	for i, sym := range cat.Symbols {
		if sym.ID == id {
			return i
		}
	}

	return -1
}

func TestDecodeTestSuite(t *testing.T) {
	suite.Run(t, new(DecodeTestSuite))
}
