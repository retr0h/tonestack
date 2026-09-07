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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// RoutingTestSuite covers turning what a device wraps a chain in into what a
// preset stores.
//
// The device names none of its own inputs and outputs, because it knows which
// they are. A file has to name them, and the catalog is where the names come
// from.
type RoutingTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *RoutingTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// entry decodes one routing slot from the result.
func (s *RoutingTestSuite) entry(
	got *map[string]json.RawMessage,
	slot string,
) map[string]any {
	s.Require().NotNil(got)

	raw, ok := (*got)[slot]
	s.Require().True(ok, "no %s", slot)

	var out map[string]any
	s.Require().NoError(json.Unmarshal(raw, &out))

	return out
}

func (s *RoutingTestSuite) TestNamesAnInputTheDeviceDidNot() {
	got := routingOf(wire.DevicePreset{Routing: []wire.DeviceRouting{{
		Slot: "inputA", Select: 1, HasSelect: true,
		Values: []any{false, -48.0, 0.5},
	}}}, s.cat)

	in := s.entry(got, "dsp0.inputA")
	s.Require().Equal("HelixStomp_AppDSPFlowInput", in["@model"])
	s.Require().InDelta(1.0, in["@input"], 0.001)

	// Named by position, from the same table that names a block's parameters.
	s.Require().Equal(false, in["noiseGate"])
	s.Require().InDelta(-48.0, in["threshold"], 0.001)
	s.Require().InDelta(0.5, in["decay"], 0.001)
}

func (s *RoutingTestSuite) TestNamesEachOutput() {
	got := routingOf(wire.DevicePreset{Routing: []wire.DeviceRouting{
		{Slot: "outputA", Select: 1, HasSelect: true, Values: []any{0.5, 0.0}},
		{Slot: "outputB", Select: 0, HasSelect: true},
	}}, s.cat)

	s.Require().Equal("HelixStomp_AppDSPFlowOutputMain",
		s.entry(got, "dsp0.outputA")["@model"])
	s.Require().Equal("HelixStomp_AppDSPFlowOutputSend",
		s.entry(got, "dsp0.outputB")["@model"])
}

func (s *RoutingTestSuite) TestASplitNamesItself() {
	// A split and a join carry their own model number, because more than one
	// kind of split exists and the device has to say which.
	split := indexOf(s.cat, "HD2_AppDSPFlowSplitY")

	got := routingOf(wire.DevicePreset{Routing: []wire.DeviceRouting{{
		Slot: "split", Model: split, HasModel: true,
		Position: 3, Enabled: true, Values: []any{0.5, 0.5, false},
	}}}, s.cat)

	e := s.entry(got, "dsp0.split")
	s.Require().Equal("HD2_AppDSPFlowSplitY", e["@model"])
	s.Require().Equal(true, e["@enabled"])
	s.Require().InDelta(3.0, e["@position"], 0.001)
	s.Require().InDelta(0.5, e["BalanceA"], 0.001)
	s.Require().Equal(false, e["bypass"])
}

func (s *RoutingTestSuite) TestSkipsWhatItCannotName() {
	for _, tc := range []struct {
		name string
		in   wire.DeviceRouting
	}{
		{"a slot nothing recognises", wire.DeviceRouting{Slot: "elsewhere"}},
		{
			"a model the table does not reach",
			wire.DeviceRouting{Slot: "split", Model: 99999, HasModel: true},
		},
	} {
		s.Run(tc.name, func() {
			s.Require().Nil(
				routingOf(wire.DevicePreset{Routing: []wire.DeviceRouting{tc.in}}, s.cat))
		})
	}
}

func (s *RoutingTestSuite) TestADeviceThatSentNoRouting() {
	s.Require().Nil(routingOf(wire.DevicePreset{}, s.cat))
	s.Require().Nil(deviceStateOf(wire.DevicePreset{}, s.cat))
}

func (s *RoutingTestSuite) TestACatalogThatNamesNoFlow() {
	// Generated before this existed. A rig that carried half the routing
	// would rebuild into a preset that routes differently from the one it
	// came from, which is worse than carrying none.
	bare := &catalog.Catalog{DeviceID: s.cat.DeviceID}

	s.Require().Nil(routingOf(wire.DevicePreset{Routing: []wire.DeviceRouting{
		{Slot: "inputA"},
	}}, bare))
}

func (s *RoutingTestSuite) TestAModelWithNoNamesForItsValues() {
	// The values are dropped rather than guessed at, and what is known about
	// the slot is still recorded.
	bare := &catalog.Catalog{
		DeviceID: s.cat.DeviceID,
		Flow:     catalog.Flow{Input: "HD2_Unknown"},
	}

	e := s.entry(routingOf(wire.DevicePreset{Routing: []wire.DeviceRouting{
		{Slot: "inputA", Values: []any{1.0}},
	}}, bare), "dsp0.inputA")

	s.Require().Equal("HD2_Unknown", e["@model"])
	s.Require().Len(e, 1)
}

func (s *RoutingTestSuite) TestRecordsWhichDeviceAnswered() {
	got := deviceStateOf(wire.DevicePreset{Routing: []wire.DeviceRouting{
		{Slot: "inputA"},
	}}, s.cat)

	s.Require().NotNil(got)
	s.Require().Equal(s.cat.DeviceID, *got.Id)
}

func TestRoutingTestSuite(t *testing.T) {
	suite.Run(t, new(RoutingTestSuite))
}
