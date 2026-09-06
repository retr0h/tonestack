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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/rig"
)

type RigPublicTestSuite struct {
	suite.Suite
}

func (s *RigPublicTestSuite) TestSpecRoundTripsThroughJSON() {
	in := rig.Spec{
		Name:   "Test Rig",
		Origin: rig.OriginCurated,
		Blocks: []rig.SpecBlock{
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
		Snapshots: []rig.Snapshot{
			{
				Name: "Lead",
				Overrides: map[int]map[string]catalog.ParamValue{
					0: {"Gain": catalog.Float(0.9)},
				},
			},
		},
	}

	b, err := json.Marshal(in)
	s.Require().NoError(err)

	var out rig.Spec
	s.Require().NoError(json.Unmarshal(b, &out))
	s.Require().Equal(in, out)
}

func (s *RigPublicTestSuite) TestHXStompLimitsAreTheDocumentedCeilings() {
	l := rig.HXStompLimits()

	s.Require().Equal(6, l.MaxBlocks)
	s.Require().Equal(2, l.Chips)
	s.Require().InDelta(0.95, l.ChipCeiling, 1e-9)
}

func (s *RigPublicTestSuite) TestOriginsAreDistinct() {
	s.Require().NotEqual(rig.OriginCurated, rig.OriginLLM)
	s.Require().NotEqual(rig.OriginLLM, rig.OriginAudio)
	s.Require().NotEqual(rig.OriginCurated, rig.OriginAudio)
}

func TestRigPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RigPublicTestSuite))
}
