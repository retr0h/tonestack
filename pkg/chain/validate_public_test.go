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

type ValidatePublicTestSuite struct {
	suite.Suite
}

func (*ValidatePublicTestSuite) limits() chain.Limits {
	return chain.Limits{MaxBlocks: 6, Paths: 2, ChipCeiling: 95.0}
}

func (s *ValidatePublicTestSuite) TestAcceptsAValidRig() {
	spec := chain.Chain{
		Name: "Fine",
		Blocks: []chain.Block{{
			Model:   "HD2_AmpTest",
			Params:  map[string]catalog.ParamValue{"Gain": catalog.Float(0.5)},
			DSP:     0,
			Pos:     0,
			Enabled: true,
		}},
	}

	s.Require().NoError(chain.Validate(newCatalog(testAmp()), spec, s.limits()))
}

func (s *ValidatePublicTestSuite) TestReportsStructureBeforeParams() {
	spec := chain.Chain{Blocks: []chain.Block{{
		Model:  "HD2_Nope",
		Params: map[string]catalog.ParamValue{"Whatever": catalog.Float(99)},
		Pos:    0,
	}}}

	s.Require().ErrorIs(
		chain.Validate(newCatalog(testAmp()), spec, s.limits()),
		chain.ErrUnknownBlock,
	)
}

func (s *ValidatePublicTestSuite) TestReportsParamsBeforeTopology() {
	spec := chain.Chain{Blocks: []chain.Block{{
		Model:  "HD2_AmpTest",
		Params: map[string]catalog.ParamValue{"Gain": catalog.Float(99)},
		Pos:    3,
	}}}

	s.Require().ErrorIs(
		chain.Validate(newCatalog(testAmp()), spec, s.limits()),
		catalog.ErrBadParam,
	)
}

func (s *ValidatePublicTestSuite) TestReportsTopologyBeforeBudget() {
	blocks := make([]chain.Block, 7)
	for i := range blocks {
		blocks[i] = chain.Block{Model: "HD2_AmpTest", DSP: 0, Pos: i}
	}

	s.Require().ErrorIs(
		chain.Validate(newCatalog(testAmp()), chain.Chain{Blocks: blocks}, s.limits()),
		chain.ErrBadTopology,
	)
}

func (s *ValidatePublicTestSuite) TestReportsBudgetLast() {
	blocks := make([]chain.Block, 4)
	for i := range blocks {
		blocks[i] = chain.Block{Model: "HD2_AmpTest", DSP: 0, Pos: i}
	}

	s.Require().ErrorIs(
		chain.Validate(newCatalog(testAmp()), chain.Chain{Blocks: blocks}, s.limits()),
		chain.ErrOverBudget,
	)
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}
