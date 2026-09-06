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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/rig"
)

type ValidateTopologyPublicTestSuite struct {
	suite.Suite
}

func (*ValidateTopologyPublicTestSuite) limits() rig.Limits {
	return rig.Limits{MaxBlocks: 6, Chips: 2, ChipCeiling: 0.95}
}

func (s *ValidateTopologyPublicTestSuite) TestAcceptsContiguousPositionsPerChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 1},
		{Model: "C", DSP: 1, Pos: 0},
	}}

	s.Require().NoError(rig.ValidateTopology(spec, s.limits()))
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsAnEmptyRig() {
	err := rig.ValidateTopology(rig.Spec{}, s.limits())

	s.Require().ErrorIs(err, rig.ErrBadTopology)
	s.Require().Contains(err.Error(), "no blocks")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsTooManyBlocks() {
	blocks := make([]rig.SpecBlock, 7)
	for i := range blocks {
		blocks[i] = rig.SpecBlock{Model: "A", DSP: 0, Pos: i}
	}

	err := rig.ValidateTopology(rig.Spec{Blocks: blocks}, s.limits())

	s.Require().ErrorIs(err, rig.ErrBadTopology)
	s.Require().Contains(err.Error(), "7 blocks")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsDuplicatePositionsOnOneChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 0},
	}}

	s.Require().
		ErrorIs(rig.ValidateTopology(spec, s.limits()), rig.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsAGapInPositions() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 2},
	}}

	s.Require().
		ErrorIs(rig.ValidateTopology(spec, s.limits()), rig.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsANegativePosition() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "A", DSP: 0, Pos: -1}}}

	s.Require().
		ErrorIs(rig.ValidateTopology(spec, s.limits()), rig.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsABlockOnANonexistentChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "A", DSP: 9, Pos: 0}}}

	s.Require().
		ErrorIs(rig.ValidateTopology(spec, s.limits()), rig.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsASnapshotOverridingAMissingBlock() {
	spec := rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []rig.Snapshot{{
			Name: "Lead",
			Overrides: map[int]map[string]catalog.ParamValue{
				4: {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	err := rig.ValidateTopology(spec, s.limits())

	s.Require().ErrorIs(err, rig.ErrBadTopology)
	s.Require().Contains(err.Error(), "snapshot")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsASnapshotWithANegativeIndex() {
	spec := rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []rig.Snapshot{{
			Name: "Lead",
			Overrides: map[int]map[string]catalog.ParamValue{
				-1: {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	s.Require().
		ErrorIs(rig.ValidateTopology(spec, s.limits()), rig.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestAcceptsAValidSnapshot() {
	spec := rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []rig.Snapshot{{
			Name: "Lead",
			Overrides: map[int]map[string]catalog.ParamValue{
				0: {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	s.Require().NoError(rig.ValidateTopology(spec, s.limits()))
}

func TestValidateTopologyPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateTopologyPublicTestSuite))
}
