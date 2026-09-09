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

// TestValidate composes the four layers, and the order they report in is
// what this covers.
//
// Each row is wrong in more than one way at once, and names which complaint
// has to come first. The order is what makes a failure actionable: being told
// a block is over budget is no use when the model does not exist.
func (s *ValidatePublicTestSuite) TestValidate() {
	crowded := make([]chain.Block, 7)
	for i := range crowded {
		crowded[i] = chain.Block{Model: "HD2_AmpTest", DSP: 0, Pos: i}
	}

	overBudget := make([]chain.Block, 4)
	for i := range overBudget {
		overBudget[i] = chain.Block{Model: "HD2_AmpTest", DSP: 0, Pos: i}
	}

	tests := []struct {
		name string
		spec chain.Chain
		is   error
	}{
		{
			name: "a rig with nothing wrong with it",
			spec: chain.Chain{
				Name: "Fine",
				Blocks: []chain.Block{{
					Model:   "HD2_AmpTest",
					Params:  map[string]catalog.ParamValue{"Gain": catalog.Float(0.5)},
					Enabled: true,
				}},
			},
		},
		{
			name: "an unknown model with a bad parameter: structure first",
			spec: chain.Chain{Blocks: []chain.Block{{
				Model:  "HD2_Nope",
				Params: map[string]catalog.ParamValue{"Whatever": catalog.Float(99)},
			}}},
			is: chain.ErrUnknownBlock,
		},
		{
			name: "a bad parameter at a bad position: parameters next",
			spec: chain.Chain{Blocks: []chain.Block{{
				Model:  "HD2_AmpTest",
				Params: map[string]catalog.ParamValue{"Gain": catalog.Float(99)},
				Pos:    3,
			}}},
			is: catalog.ErrBadParam,
		},
		{
			name: "too many blocks, which are also too expensive: topology next",
			spec: chain.Chain{Blocks: crowded},
			is:   chain.ErrBadTopology,
		},
		{
			name: "a chain that is only too expensive: budget last",
			spec: chain.Chain{Blocks: overBudget},
			is:   chain.ErrOverBudget,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := chain.Validate(newCatalog(testAmp()), tt.spec, s.limits())

			if tt.is == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.is)
		})
	}
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}
