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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type MatchPublicTestSuite struct {
	suite.Suite
}

func (s *MatchPublicTestSuite) TestMatches() {
	svt := catalog.Block{Name: "Ampeg SVT Nrm", BasedOn: "Ampeg SVT® (normal channel)"}
	original := catalog.Block{Name: "Line 6 Litigator", BasedOn: "Line 6 Original"}
	klon := catalog.Block{Name: "Minotaur", BasedOn: "Klon® Centaur"}

	tests := []struct {
		name  string
		block catalog.Block
		want  string
		match bool
	}{
		{"the gear a block emulates", svt, "Ampeg SVT", true},
		{
			"a trademark symbol nobody types",
			klon, "Klon Centaur", true,
		},
		{
			"a model's own name, which is the only handle a Line 6 original has",
			original, "Line 6 Litigator", true,
		},
		{"a name in a different case", svt, "ampeg svt", true},
		{"a name with the spacing off", svt, "  Ampeg   SVT  ", true},
		{"part of a name", klon, "Minotaur", true},
		{"gear this is not", svt, "Marshall JCM800", false},
		{"nothing at all", svt, "", false},
		{"nothing but spaces", svt, "   ", false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.match, tc.block.Matches(tc.want))
		})
	}
}

func (s *MatchPublicTestSuite) TestAModelThatEmulatesNothingStillMatchesByName() {
	// Every Line 6 original reads "Line 6 Original" as the gear it is based
	// on, so matching that alone would make all of them impossible to ask for.
	b := catalog.Block{Name: "Glitz", BasedOn: "Line 6 Original"}

	s.Require().True(b.Matches("Glitz"))
	s.Require().True(b.Matches("Line 6 Original"))
}

func TestMatchPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MatchPublicTestSuite))
}
