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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
)

type ValidateBudgetPublicTestSuite struct {
	suite.Suite
}

func (*ValidateBudgetPublicTestSuite) limits() chain.Limits {
	return chain.Limits{MaxBlocks: 6, Paths: 2, ChipCeiling: 95.0}
}

func (s *ValidateBudgetPublicTestSuite) TestAcceptsARigInsideTheCeiling() {
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 1, Enabled: true},
	}}

	s.Require().
		NoError(chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits()))
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsAChipOverTheCeiling() {
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
	}}

	err := chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits())

	s.Require().ErrorIs(err, chain.ErrOverBudget)

	var target *chain.OverBudgetError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal(0, target.Chip)
	s.Require().InDelta(106.68, target.Cost, 1e-9)
}

func (s *ValidateBudgetPublicTestSuite) TestCountsBypassedBlocks() {
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
	}}

	s.Require().ErrorIs(
		chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		chain.ErrOverBudget,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestChargesStereoBlocksTheStereoCost() {
	// stereo costs 40.1 each; three is 120.3, over 95
	blk := testAmp()
	blk.Stereo = true

	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
	}}

	// three stereo instances cost 120.3; three mono would be 80.01 and fit
	s.Require().ErrorIs(
		chain.ValidateBudget(newCatalog(blk), spec, s.limits()),
		chain.ErrOverBudget,
	)
	s.Require().NoError(
		chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		"the same three blocks fit when mono")
}

func (s *ValidateBudgetPublicTestSuite) TestRefusesAnAssumedDSPCost() {
	blk := testAmp()
	blk.DSP.Prov = catalog.ProvAssumed

	spec := chain.Chain{Blocks: []chain.Block{{Model: "HD2_AmpTest", DSP: 0}}}

	err := chain.ValidateBudget(newCatalog(blk), spec, s.limits())

	s.Require().ErrorIs(err, catalog.ErrBadParam)
	s.Require().Contains(err.Error(), "assumed")
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsABlockOnANonexistentChip() {
	spec := chain.Chain{Blocks: []chain.Block{{Model: "HD2_AmpTest", DSP: 5}}}

	s.Require().ErrorIs(
		chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		chain.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsANegativeChipIndex() {
	spec := chain.Chain{Blocks: []chain.Block{{Model: "HD2_AmpTest", DSP: -1}}}

	s.Require().ErrorIs(
		chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		chain.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsLimitsDeclaringNoChips() {
	spec := chain.Chain{Blocks: []chain.Block{{Model: "HD2_AmpTest"}}}

	s.Require().ErrorIs(
		chain.ValidateBudget(newCatalog(testAmp()), spec, chain.Limits{Paths: 0}),
		chain.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsAnUnknownModel() {
	spec := chain.Chain{Blocks: []chain.Block{{Model: "HD2_Nope"}}}

	s.Require().ErrorIs(
		chain.ValidateBudget(newCatalog(testAmp()), spec, s.limits()),
		chain.ErrUnknownBlock,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestARealisticRigFits() {
	// Regression: ChipCeiling was once a fraction while the catalog states
	// cost in percent, so every rig was rejected as over budget — including
	// a single amp. A chain the device would happily load must validate.
	spec := chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_AmpTest", DSP: 0, Pos: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Pos: 1, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Pos: 2, Enabled: true},
	}}

	s.Require().NoError(
		chain.ValidateBudget(newCatalog(testAmp()), spec, chain.HXStompLimits()),
		"three amps on the one signal path a Stomp has is well within it")
}

func (s *ValidateBudgetPublicTestSuite) TestCostAndCeilingShareUnits() {
	// The catalog states an Ampeg SVT at 26.67. A ceiling below that would
	// mean no amp ever fits, which is how the units drifted apart before.
	s.Require().Greater(chain.HXStompLimits().ChipCeiling, 26.67,
		"the ceiling must admit at least one amp")
}

func TestValidateBudgetPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateBudgetPublicTestSuite))
}
