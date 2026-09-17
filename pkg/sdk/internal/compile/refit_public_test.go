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
package compile_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// RefitPublicTestSuite covers assignments following the blocks they name.
type RefitPublicTestSuite struct {
	suite.Suite
}

// TestRefit covers where an assignment ends up after a chain is laid out.
func (s *RefitPublicTestSuite) TestRefit() {
	// Three blocks, the last of which the fit put on the second processor
	// and numbered from zero again.
	before := []chain.Block{{Pos: 0}, {Pos: 1}, {Pos: 2}}
	after := []chain.Block{{Pos: 0}, {Pos: 1}, {DSP: 1, Pos: 0}}

	tests := []struct {
		name          string
		control       *[]rig.Controller
		before, after []chain.Block
		wantPath      int
		wantBlock     int
	}{
		{
			name:    "a block the fit moved",
			control: &[]rig.Controller{{Controller: 2, Block: 2, Parameter: "Mix"}},
			before:  before, after: after,
			wantPath: 1, wantBlock: 0,
		},
		{
			name:    "a block the fit left alone",
			control: &[]rig.Controller{{Controller: 2, Block: 1, Parameter: "Mix"}},
			before:  before, after: after,
			wantPath: 0, wantBlock: 1,
		},
		{
			// Nothing to move it onto, so it says what it said. Whether that
			// is a block the chain has is check's answer, not this one.
			name:    "a block the chain never had",
			control: &[]rig.Controller{{Controller: 2, Block: 9, Parameter: "Mix"}},
			before:  before, after: after,
			wantPath: 0, wantBlock: 9,
		},
		{
			// Two lists that are not the same chain say nothing about where
			// anything went.
			name:    "a chain that does not line up",
			control: &[]rig.Controller{{Controller: 2, Block: 2, Parameter: "Mix"}},
			before:  before, after: after[:2],
			wantPath: 0, wantBlock: 2,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := compile.Refit(
				rig.Spec{Controllers: tt.control}, tt.before, tt.after)

			s.Require().NotNil(got.Controllers)
			s.Require().Len(*got.Controllers, 1)

			one := (*got.Controllers)[0]
			s.Require().Equal(tt.wantBlock, one.Block)

			path := 0
			if one.Path != nil {
				path = *one.Path
			}

			s.Require().Equal(tt.wantPath, path)
		})
	}
}

// TestRefitMovesAFootswitch covers a switch following its block, which is the
// same numbers and the same fit.
func (s *RefitPublicTestSuite) TestRefitMovesAFootswitch() {
	before := []chain.Block{{Pos: 0}, {Pos: 1}, {Pos: 2}}
	after := []chain.Block{{Pos: 0}, {Pos: 1}, {DSP: 1, Pos: 0}}

	on, at, nowhere := 1, 2, 9
	spec := rig.Spec{Footswitches: &[]rig.Footswitch{
		{Switch: &on, Block: &at},
		{Switch: &on, Block: &nowhere},
		// A switch that acts on nothing has no block for the fit to move.
		{Switch: &on},
	}}

	got := compile.Refit(spec, before, after)
	s.Require().NotNil(got.Footswitches)

	moved := (*got.Footswitches)[0]
	s.Require().Equal(0, *moved.Block)
	s.Require().Equal(1, *moved.Path)

	kept := (*got.Footswitches)[1]
	s.Require().Equal(9, *kept.Block)
	s.Require().Nil(kept.Path)

	s.Require().Nil((*got.Footswitches)[2].Block)
}

// TestRefitLeavesARigThatAssignsNothing covers the common case, where there
// is nothing to move.
func (s *RefitPublicTestSuite) TestRefitLeavesARigThatAssignsNothing() {
	spec := rig.Spec{ID: "test"}

	s.Require().Equal(spec, compile.Refit(spec, nil, nil))
}

func TestRefitPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RefitPublicTestSuite))
}
