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

type ValidateBudgetPublicTestSuite struct {
	suite.Suite
}

func (*ValidateBudgetPublicTestSuite) limits() rig.Limits {
	return rig.Limits{MaxBlocks: 6, Chips: 2, ChipCeiling: 0.95}
}

func (s *ValidateBudgetPublicTestSuite) TestAcceptsARigInsideTheCeiling() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 1, Enabled: true},
	}}

	s.Require().
		NoError(rig.ValidateBudget(newCatalog(testAmp()), spec, s.limits()))
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsAChipOverTheCeiling() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
	}}

	err := rig.ValidateBudget(newCatalog(testAmp()), spec, s.limits())

	s.Require().ErrorIs(err, rig.ErrOverBudget)

	var target *rig.OverBudgetError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal(0, target.Chip)
	s.Require().InDelta(1.20, target.Cost, 1e-9)
}

func (s *ValidateBudgetPublicTestSuite) TestCountsBypassedBlocks() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
	}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		rig.ErrOverBudget,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestChargesStereoBlocksTheStereoCost() {
	blk := testAmp()
	blk.Stereo = true

	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
	}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newCatalog(blk), spec, s.limits()),
		rig.ErrOverBudget,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRefusesAnAssumedDSPCost() {
	blk := testAmp()
	blk.DSP.Prov = catalog.ProvAssumed

	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", DSP: 0}}}

	err := rig.ValidateBudget(newCatalog(blk), spec, s.limits())

	s.Require().ErrorIs(err, catalog.ErrBadParam)
	s.Require().Contains(err.Error(), "assumed")
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsABlockOnANonexistentChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", DSP: 5}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		rig.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsANegativeChipIndex() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", DSP: -1}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		rig.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsLimitsDeclaringNoChips() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest"}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newCatalog(testAmp()), spec, rig.Limits{Chips: 0}),
		rig.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsAnUnknownModel() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_Nope"}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		rig.ErrUnknownBlock,
	)
}

func TestValidateBudgetPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateBudgetPublicTestSuite))
}
