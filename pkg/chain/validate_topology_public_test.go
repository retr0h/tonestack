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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
)

type ValidateTopologyPublicTestSuite struct {
	suite.Suite
}

func (*ValidateTopologyPublicTestSuite) limits() chain.Limits {
	return chain.Limits{MaxBlocks: 6, Paths: 2, ChipCeiling: 95.0}
}

func (s *ValidateTopologyPublicTestSuite) TestAcceptsContiguousPositionsPerChip() {
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 1},
		{Model: "C", DSP: 1, Pos: 0},
	}}

	s.Require().NoError(chain.ValidateTopology(spec, s.limits()))
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsAnEmptyRig() {
	err := chain.ValidateTopology(chain.Chain{}, s.limits())

	s.Require().ErrorIs(err, chain.ErrBadTopology)
	s.Require().Contains(err.Error(), "no blocks")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsTooManyBlocks() {
	blocks := make([]chain.Block, 7)
	for i := range blocks {
		blocks[i] = chain.Block{Model: "A", DSP: 0, Pos: i}
	}

	err := chain.ValidateTopology(chain.Chain{Blocks: blocks}, s.limits())

	s.Require().ErrorIs(err, chain.ErrBadTopology)
	s.Require().Contains(err.Error(), "7 blocks")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsDuplicatePositionsOnOneChip() {
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 0},
	}}

	s.Require().
		ErrorIs(chain.ValidateTopology(spec, s.limits()), chain.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsAGapInPositions() {
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 2},
	}}

	s.Require().
		ErrorIs(chain.ValidateTopology(spec, s.limits()), chain.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsANegativePosition() {
	spec := chain.Chain{Blocks: []chain.Block{{Model: "A", DSP: 0, Pos: -1}}}

	s.Require().
		ErrorIs(chain.ValidateTopology(spec, s.limits()), chain.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsABlockOnANonexistentChip() {
	spec := chain.Chain{Blocks: []chain.Block{{Model: "A", DSP: 9, Pos: 0}}}

	s.Require().
		ErrorIs(chain.ValidateTopology(spec, s.limits()), chain.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsASnapshotOverridingAMissingBlock() {
	spec := chain.Chain{
		Blocks: []chain.Block{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []chain.Snapshot{{
			Name: "Lead",
			Overrides: map[string]chain.Params{
				"4": {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	err := chain.ValidateTopology(spec, s.limits())

	s.Require().ErrorIs(err, chain.ErrBadTopology)
	s.Require().Contains(err.Error(), "snapshot")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsASnapshotWithANegativeIndex() {
	spec := chain.Chain{
		Blocks: []chain.Block{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []chain.Snapshot{{
			Name: "Lead",
			Overrides: map[string]chain.Params{
				"-1": {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	s.Require().
		ErrorIs(chain.ValidateTopology(spec, s.limits()), chain.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestAcceptsAValidSnapshot() {
	spec := chain.Chain{
		Blocks: []chain.Block{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []chain.Snapshot{{
			Name: "Lead",
			Overrides: map[string]chain.Params{
				"0": {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	s.Require().NoError(chain.ValidateTopology(spec, s.limits()))
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsASnapshotKeyThatIsNotAnIndex() {
	// Overrides are keyed by string because JSON object keys are strings. A
	// key that is not a number cannot name a block.
	spec := chain.Chain{
		Blocks: []chain.Block{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []chain.Snapshot{{
			Name:      "Lead",
			Overrides: map[string]chain.Params{"first": {"Gain": catalog.Float(0.9)}},
		}},
	}

	err := chain.ValidateTopology(spec, s.limits())

	s.Require().ErrorIs(err, chain.ErrBadTopology)
	s.Require().Contains(err.Error(), "not an index")
}

func TestValidateTopologyPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateTopologyPublicTestSuite))
}
