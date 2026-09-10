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

package editor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
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

// TestRoutingOf names what the device keeps to itself.
func (s *RoutingTestSuite) TestRoutingOf() {
	amp := indexOf(s.cat, "HD2_AmpTucknGo")

	tests := []struct {
		name    string
		routing []wire.DeviceRouting
		blocks  []wire.DeviceBlock
		// a catalog with no flow models, or one naming a model whose
		// parameters have no names.
		bare    bool
		unnamed bool

		// what each slot must carry, must not carry, and how many fields it
		// holds in all.
		want   map[string]map[string]any
		absent map[string][]string
		sizes  map[string]int
		// nothing worth recording.
		none bool
	}{
		{
			name: "an input the device did not name",
			routing: []wire.DeviceRouting{{
				Slot: "inputA", Select: 1, HasSelect: true,
				Values: []any{false, -48.0, 0.5},
			}},
			want: map[string]map[string]any{"dsp0.inputA": {
				"@model": "HelixStomp_AppDSPFlowInput",
				"@input": 1.0,
				// Named by position, from the same table that names a block's
				// parameters.
				"noiseGate": false,
				"threshold": -48.0,
				"decay":     0.5,
			}},
		},
		{
			name: "each output",
			routing: []wire.DeviceRouting{
				{Slot: "outputA", Select: 1, HasSelect: true, Values: []any{0.5, 0.0}},
				{Slot: "outputB", Select: 0, HasSelect: true},
			},
			want: map[string]map[string]any{
				"dsp0.outputA": {"@model": "HelixStomp_AppDSPFlowOutputMain"},
				"dsp0.outputB": {"@model": "HelixStomp_AppDSPFlowOutputSend"},
			},
		},
		{
			// A split and a join carry their own model number, because more
			// than one kind of split exists and the device has to say which.
			name: "a split, which names itself",
			routing: []wire.DeviceRouting{{
				Slot: "split", Model: indexOf(s.cat, "HD2_AppDSPFlowSplitY"),
				HasModel: true,
				Position: 3, Enabled: true, Values: []any{0.5, 0.5, false},
			}},
			want: map[string]map[string]any{"dsp0.split": {
				"@model":    "HD2_AppDSPFlowSplitY",
				"@enabled":  true,
				"@position": 3.0,
				"BalanceA":  0.5,
				"bypass":    false,
			}},
		},
		{
			// The values are dropped rather than guessed at, and what is
			// known about the slot is still recorded.
			name:    "a model with no names for its values",
			routing: []wire.DeviceRouting{{Slot: "inputA", Values: []any{1.0}}},
			unnamed: true,
			want:    map[string]map[string]any{"dsp0.inputA": {"@model": "HD2_Unknown"}},
			sizes:   map[string]int{"dsp0.inputA": 1},
		},
		{
			// A device stores an amp and its cabinet as one block. A preset
			// stores the amp with a `@cab` and the cabinet as a sibling, and
			// 304 of 721 HX Stomp presets in the corpus have one.
			name: "a cabinet an amplifier carries",
			blocks: []wire.DeviceBlock{{
				Model: amp, CabNamed: 5,
				Cab: []any{3.0, 20.0, 15000.0, 0.2, 2.0, int64(10)},
			}},
			want: map[string]map[string]any{"dsp0.cab0": {
				"@model": "HD2_Cab1x15TucknGo", "@enabled": true, "@mic": 10.0,
				"Distance": 3.0, "LowCut": 20.0, "HighCut": 15000.0,
				"EarlyReflections": 0.2, "Level": 2.0,
			}},
		},
		{
			// Anything past what the model has names for is the microphone. A
			// device that sent no more than the names says nothing about one.
			name: "a cabinet with no microphone reported",
			blocks: []wire.DeviceBlock{{
				Model: amp, CabNamed: 5,
				Cab: []any{3.0, 20.0, 15000.0, 0.2, 2.0},
			}},
			want: map[string]map[string]any{"dsp0.cab0": {
				"@model": "HD2_Cab1x15TucknGo", "Distance": 3.0,
			}},
			absent: map[string][]string{"dsp0.cab0": {"@mic"}},
		},
		{
			name:    "a slot nothing recognises",
			routing: []wire.DeviceRouting{{Slot: "elsewhere"}},
			none:    true,
		},
		{
			name: "a routing model the table does not reach",
			routing: []wire.DeviceRouting{
				{Slot: "split", Model: 99999, HasModel: true},
			},
			none: true,
		},
		{name: "a device that sent no routing at all", none: true},
		{
			// Generated before this existed. A rig that carried half the
			// routing would rebuild into a preset that routes differently
			// from the one it came from, which is worse than carrying none.
			name:    "a catalog that names no flow",
			routing: []wire.DeviceRouting{{Slot: "inputA"}},
			bare:    true,
			none:    true,
		},
		{
			name:   "a block carrying no cabinet",
			blocks: []wire.DeviceBlock{{Model: amp}},
			none:   true,
		},
		{
			name:   "a cabinet on a model the table does not reach",
			blocks: []wire.DeviceBlock{{Model: 99999, Cab: []any{1.0}}},
			none:   true,
		},
		{
			name: "a model that names no pairing",
			blocks: []wire.DeviceBlock{{
				Model: indexOf(s.cat, "HD2_DistTeemahMono"), Cab: []any{1.0},
			}},
			none: true,
		},
		{
			// Symbols cover every Helix; a Stomp has no second effects loop.
			name: "hardware this device does not have",
			blocks: []wire.DeviceBlock{{
				Model: indexOf(s.cat, "HD2_FXLoopMono3"), Cab: []any{1.0},
			}},
			none: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cat := s.cat

			switch {
			case tt.bare:
				cat = &catalog.Catalog{DeviceID: s.cat.DeviceID}
			case tt.unnamed:
				cat = &catalog.Catalog{
					DeviceID: s.cat.DeviceID,
					Flow:     catalog.Flow{Input: "HD2_Unknown"},
				}
			}

			got := routingOf(
				wire.DevicePreset{Routing: tt.routing, Blocks: tt.blocks}, cat)

			if tt.none {
				s.Require().Nil(got)

				return
			}

			for slot, want := range tt.want {
				e := s.entry(got, slot)

				for key, value := range want {
					s.Require().Equal(value, e[key], "%s of %s", key, slot)
				}

				for _, key := range tt.absent[slot] {
					s.Require().NotContains(e, key)
				}

				if size, ok := tt.sizes[slot]; ok {
					s.Require().Len(e, size)
				}
			}
		})
	}
}

// TestDeviceStateOf records which device answered.
func (s *RoutingTestSuite) TestDeviceStateOf() {
	tests := []struct {
		name    string
		routing []wire.DeviceRouting
		want    bool
	}{
		{
			name:    "a device that sent its routing",
			routing: []wire.DeviceRouting{{Slot: "inputA"}},
			want:    true,
		},
		{name: "one that sent none"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := DeviceState(wire.DevicePreset{Routing: tt.routing}, s.cat)

			if !tt.want {
				s.Require().Nil(got)

				return
			}

			s.Require().NotNil(got)
			s.Require().Equal(s.cat.DeviceID, *got.Id)
		})
	}
}

func TestRoutingTestSuite(t *testing.T) {
	suite.Run(t, new(RoutingTestSuite))
}
