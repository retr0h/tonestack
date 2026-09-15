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

package result_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// ScaffoldedPublicTestSuite covers what a caller is handed for a new recipe.
type ScaffoldedPublicTestSuite struct {
	suite.Suite
}

// TestCopied covers telling a copy from a rig scaffolded out of gear names.
func (s *ScaffoldedPublicTestSuite) TestCopied() {
	tests := []struct {
		name string
		in   result.Scaffolded
		want bool
	}{
		{
			// A copy holds the parent's chain, so nothing about its gear was
			// resolved. What it was copied from is the only record of that.
			name: "a rig copied from another",
			in:   result.Scaffolded{ID: "flea-live", From: "flea"},
			want: true,
		},
		{
			name: "a rig scaffolded from gear names",
			in:   result.Scaffolded{ID: "my-rig"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Copied())
		})
	}
}

func TestScaffoldedPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ScaffoldedPublicTestSuite))
}
