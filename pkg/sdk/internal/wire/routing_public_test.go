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

package wire_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// RoutingPublicTestSuite covers writing somebody else's routing into a preset.
//
// Checked by reading the result back with the decoder the rest of this project
// uses. A device wraps a chain in six entries and a chain written without them
// routes the way the preset it was built into routed, not the way the file
// says.
type RoutingPublicTestSuite struct {
	suite.Suite
}

func (s *RoutingPublicTestSuite) blank() *wire.Document {
	doc, err := wire.Blank()
	s.Require().NoError(err)

	return doc
}

// read decodes a document the way a device's answer is read.
func (s *RoutingPublicTestSuite) read(
	doc *wire.Document,
) wire.DevicePreset {
	got, err := wire.DecodePreset(doc.Encode())
	s.Require().NoError(err)

	return got
}

// slot finds one routing entry in what came back.
func (s *RoutingPublicTestSuite) slot(
	got wire.DevicePreset,
	name string,
) wire.DeviceRouting {
	for _, r := range got.Routing {
		if r.Slot == name {
			return r
		}
	}

	s.Require().FailNow("no routing slot " + name)

	return wire.DeviceRouting{}
}

// TestAnOutputTakesItsOwnValues covers the case the task was filed for.
//
// A preset whose output gain is -2.9 arrived at 0, because the gain came from
// the preset being written into rather than from the file.
func (s *RoutingPublicTestSuite) TestAnOutputTakesItsOwnValues() {
	doc := s.blank()

	before := s.slot(s.read(doc), "outputA")
	s.Require().Equal([]any{0.5, 0.0}, before.Values, "the blank's own")

	selects := 0
	values := []any{0.25, -2.9000000953674316}

	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "outputA", Select: &selects, Values: &values, Named: 2},
	}))

	got := s.slot(s.read(doc), "outputA")

	s.Require().Equal(0, got.Select)
	s.Require().Len(got.Values, 2)
	s.Require().InDelta(0.25, got.Values[0], 0.0001)
	s.Require().InDelta(-2.9, got.Values[1], 0.0001)
}

// TestASplitMovesAndSwitchesOff covers the fields only a split and a join have.
func (s *RoutingPublicTestSuite) TestASplitMovesAndSwitchesOff() {
	doc := s.blank()

	off := false
	position := 7

	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "split", Enabled: &off, Position: &position},
	}))

	got := s.slot(s.read(doc), "split")

	s.Require().False(got.Enabled)
	s.Require().Equal(7, got.Position)
	s.Require().Equal(257, got.Model, "the model nobody changed")
	s.Require().Equal([]any{0.5, 0.5, false}, got.Values, "and its values")
}

// TestAJoinTakesEverySetting covers the longest value list a device sends.
func (s *RoutingPublicTestSuite) TestAJoinTakesEverySetting() {
	doc := s.blank()

	values := []any{0.1, 0.2, 0.3, 0.4, true, 5.0}

	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "join", Values: &values, Named: 6},
	}))

	got := s.slot(s.read(doc), "join")

	s.Require().Len(got.Values, 6)
	s.Require().InDelta(0.1, got.Values[0], 0.0001)
	s.Require().Equal(true, got.Values[4])
	s.Require().InDelta(5, got.Values[5], 0.0001)
}

// TestWhatNobodyNamesIsLeftAlone is what makes every field optional.
func (s *RoutingPublicTestSuite) TestWhatNobodyNamesIsLeftAlone() {
	doc := s.blank()

	before := s.slot(s.read(doc), "inputA")

	position := 3
	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "split", Position: &position},
	}))

	got := s.slot(s.read(doc), "inputA")

	s.Require().Equal(before.Values, got.Values)
	s.Require().Equal(before.Select, got.Select)
}

// TestBlankRouting is where the value lengths come from.
//
// Anything building a preset into the blank needs to know how many values
// each entry takes, because a device sends fewer than a model names.
func (s *RoutingPublicTestSuite) TestBlankRouting() {
	got := wire.BlankRouting()

	s.Require().Len(got, 6)

	by := map[string]wire.DeviceRouting{}
	for _, r := range got {
		by[r.Slot] = r
	}

	s.Require().Len(by["inputA"].Values, 3, "an input sends three of seven")
	s.Require().Len(by["outputA"].Values, 2, "an output sends two of three")
	s.Require().True(by["split"].HasModel)
	s.Require().True(by["join"].HasModel)
	s.Require().False(by["inputA"].HasModel, "a device knows its own input")
}

// TestASplitTakesAnotherModel covers a file naming a split the blank has not.
//
// The parameter list changes length with the model, so the whole map is
// replaced rather than the values inside it.
func (s *RoutingPublicTestSuite) TestASplitTakesAnotherModel() {
	doc := s.blank()

	model := 151
	values := []any{0.1, 0.2, 0.3, 0.4, false, 1.0}

	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "split", Model: &model, Values: &values, Named: 6},
	}))

	got := s.slot(s.read(doc), "split")

	s.Require().Equal(151, got.Model)
	s.Require().Len(got.Values, 6, "the new model's parameters, not the old's")
}

// TestASlotNothingKnows is ignored rather than reported.
func (s *RoutingPublicTestSuite) TestASlotNothingKnows() {
	doc := s.blank()
	before := doc.Encode()

	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{{Slot: "nonsense"}}))
	s.Require().Equal(before, doc.Encode())
}

// TestAValueThatWillNotEncode is the caller's mistake, and is reported.
func (s *RoutingPublicTestSuite) TestAValueThatWillNotEncode() {
	doc := s.blank()

	values := []any{make(chan int)}
	err := wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "join", Values: &values, Named: 1},
	})

	s.Require().ErrorIs(err, wire.ErrBadValue)
}

// TestNoRouting changes nothing.
func (s *RoutingPublicTestSuite) TestNoRouting() {
	doc := s.blank()
	before := doc.Encode()

	s.Require().NoError(wire.PlaceRouting(doc, nil))
	s.Require().Equal(before, doc.Encode())
}

// TestAPresetWithNoChain has nowhere to put any of it.
func (s *RoutingPublicTestSuite) TestAPresetWithNoChain() {
	doc := wire.NewDocument(s.blank(), []int8{wire.KeySnapshots})
	before := doc.Encode()

	position := 3
	s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{
		{Slot: "split", Position: &position},
	}))

	s.Require().Equal(before, doc.Encode())
}

// replace swaps what a path holds, so a chain no device writes can be built.
func (s *RoutingPublicTestSuite) replace(
	body []byte,
	at wire.Path,
	raw []byte,
) []byte {
	start, end, err := wire.Locate(body, at)
	s.Require().NoError(err)

	out := append([]byte{}, body[:start]...)
	out = append(out, raw...)

	return append(out, body[end:]...)
}

// TestRoutingOnAChainTheDeviceDidNotWrite covers grids no device produces.
//
// Each one is reached by rebuilding the chain rather than by asking a capture
// for something it does not have, the same way place.go's own tests do it.
// None of these can arrive from hardware, and a writer that trusted the shape
// would splice bytes into the middle of somebody's preset.
func (s *RoutingPublicTestSuite) TestRoutingOnAChainTheDeviceDidNotWrite() {
	values := []any{1.0}

	tests := []struct {
		name  string
		chain []byte
		route wire.Routing
	}{
		{
			name:  "a grid shorter than the device lays out",
			chain: []byte{0x91, 0x82, 0x13, 0x08, 0x14, 0xc0},
			route: wire.Routing{Slot: "split", Values: &values, Named: 1},
		},
		{
			name:  "a position that is not a map",
			chain: []byte{0x91, 0x2a},
			route: wire.Routing{Slot: "split"},
		},
		{
			name:  "a position not saying what kind it is",
			chain: []byte{0x91, 0x81, 0x14, 0xc0},
			route: wire.Routing{Slot: "split"},
		},
		{
			name:  "a kind that is not a number",
			chain: []byte{0x91, 0x82, 0x13, 0xa1, 0x61, 0x14, 0xc0},
			route: wire.Routing{Slot: "split"},
		},
		{
			// An input that says what it is and then carries no parameters
			// at all, so there is nothing to put values into.
			name:  "an entry keeping no parameters",
			chain: []byte{0x91, 0x82, 0x13, 0x00, 0x14, 0x81, 0x05, 0x00},
			route: wire.Routing{Slot: "inputA", Values: &values, Named: 1},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()

			body, ok := doc.Section(wire.KeyTone)
			s.Require().True(ok)

			doc.SetSection(wire.KeyTone, s.replace(body, wire.Path{22}, tt.chain))

			s.Require().NoError(wire.PlaceRouting(doc, []wire.Routing{tt.route}))
		})
	}
}

func TestRoutingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RoutingPublicTestSuite))
}
