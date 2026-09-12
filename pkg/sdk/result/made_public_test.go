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

// MadePublicTestSuite covers what a caller is handed for a build.
type MadePublicTestSuite struct {
	suite.Suite
}

// TestActed covers telling a word that turned a knob from one that did not.
func (s *MadePublicTestSuite) TestActed() {
	tests := []struct {
		name string
		in   result.Moved
		want bool
	}{
		{
			name: "a word that moved a control",
			in:   result.Moved{Term: "mid-forward", Param: "Mid", From: 0.5, To: 0.58},
			want: true,
		},
		{
			// Most of the vocabulary. Six of the ten axes are not amplifier
			// controls, and a term from one of those is recorded and moves
			// nothing.
			name: "a word nothing acts on yet",
			in:   result.Moved{Term: "glassy"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Acted())
		})
	}
}

// TestContested covers a rig answering one question twice.
func (s *MadePublicTestSuite) TestContested() {
	tests := []struct {
		name string
		in   result.Moved
		want bool
	}{
		{
			// Two words from one axis cancel, so neither is applied and the
			// axis is named instead.
			name: "another word already spoke for this axis",
			in:   result.Moved{Term: "minimal-drive", Against: "drive"},
			want: true,
		},
		{
			name: "the only word on its axis",
			in:   result.Moved{Term: "mid-forward", Param: "Mid"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Contested())
		})
	}
}

func TestMadePublicTestSuite(t *testing.T) {
	suite.Run(t, new(MadePublicTestSuite))
}
