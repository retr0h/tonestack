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

package chain_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

type RigPublicTestSuite struct {
	suite.Suite
}

func (s *RigPublicTestSuite) TestSpecRoundTripsThroughJSON() {
	in := chain.Chain{
		Name: "Test Rig",
		Blocks: []chain.Block{
			{
				Model: "HD2_AmpTest",
				Params: map[string]catalog.ParamValue{
					"Gain": catalog.Float(0.5),
				},
				DSP:     0,
				Pos:     0,
				Enabled: true,
			},
		},
		Snapshots: []chain.Snapshot{
			{
				Name: "Lead",
				Overrides: map[string]chain.Params{
					"0": {"Gain": catalog.Float(0.9)},
				},
			},
		},
	}

	b, err := json.Marshal(in)
	s.Require().NoError(err)

	var out chain.Chain
	s.Require().NoError(json.Unmarshal(b, &out))
	s.Require().Equal(in, out)
}

// TestLimitsForNamesTheDevice covers each device's ceilings being its own.
func (s *RigPublicTestSuite) TestLimitsForNamesTheDevice() {
	tests := []struct {
		name   string
		device string
		blocks int
		paths  int
		why    string
	}{
		{
			name: "an HX Stomp", device: "HX Stomp", blocks: 8, paths: 1,
			why: "721 presets, the largest holding eight blocks, none on a second processor",
		},
		{
			name: "an HX Stomp XL", device: "HX Stomp XL", blocks: 8, paths: 1,
			why: "the same two chips in a bigger box",
		},
		{
			name: "a Helix Floor", device: "Helix Floor", blocks: 29, paths: 2,
			why: "1,698 presets, the largest holding 29 blocks, 1,219 on two processors",
		},
		{
			name: "a Helix LT", device: "Helix LT", blocks: 29, paths: 2,
			why: "the Helix Floor's processing in a smaller box",
		},
		{
			name: "a device named the way somebody typed it", device: "helix floor",
			blocks: 29, paths: 2, why: "a catalog's name is not case",
		},
		{
			name: "a device nothing here knows", device: "Pod Go", blocks: 8, paths: 1,
			why: "the smallest ceilings here, because a chain that fits them fits the rest",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			l := chain.LimitsFor(tt.device)

			s.Require().Equal(tt.blocks, l.MaxBlocks, tt.why)
			s.Require().Equal(tt.paths, l.Paths, tt.why)
			s.Require().InDelta(95.0, l.ChipCeiling, 1e-9)
		})
	}
}

func (s *RigPublicTestSuite) TestHXStompLimitsAreTheDocumentedCeilings() {
	l := chain.HXStompLimits()

	s.Require().Equal(8, l.MaxBlocks,
		"the corpus shows HX Stomp presets holding eight blocks")
	s.Require().Equal(1, l.Paths,
		"no HX Stomp preset in the corpus has a second signal path")
	s.Require().InDelta(95.0, l.ChipCeiling, 1e-9)
	s.Require().Greater(l.ChipCeiling, 1.0,
		"the ceiling is a percentage, matching how Line 6 states a block's cost")
}

func TestRigPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RigPublicTestSuite))
}
