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

	"github.com/retr0h/tonestack/pkg/rig"
)

type ValidateStructurePublicTestSuite struct {
	suite.Suite
}

func (s *ValidateStructurePublicTestSuite) TestAcceptsARigOfKnownBlocks() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest"}}}

	s.Require().NoError(rig.ValidateStructure(newCatalog(testAmp()), spec))
}

func (s *ValidateStructurePublicTestSuite) TestAcceptsAnEmptyRig() {
	s.Require().NoError(rig.ValidateStructure(newCatalog(), rig.Spec{}))
}

func (s *ValidateStructurePublicTestSuite) TestRejectsAnUnknownModel() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest"},
		{Model: "HD2_Nope"},
	}}

	err := rig.ValidateStructure(newCatalog(testAmp()), spec)

	s.Require().ErrorIs(err, rig.ErrUnknownBlock)

	var target *rig.UnknownBlockError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal("HD2_Nope", target.Model)
}

func TestValidateStructurePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateStructurePublicTestSuite))
}
