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
			// Four of the ten axes describe the player and the instrument
			// rather than the rig, and a term from one of those is recorded
			// and moves nothing.
			name: "a word nothing acts on yet",
			in:   result.Moved{Term: "short-decay"},
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

// TestUnanswered covers telling the two silences apart.
func (s *MadePublicTestSuite) TestUnanswered() {
	tests := []struct {
		name string
		in   result.Moved
		want bool
	}{
		{
			// A control exists for this word. This chain does not have it,
			// which is a fact about the rig and worth saying.
			name: "the chain has nowhere to put it",
			in: result.Moved{
				Term:    "audible-pick-attack",
				Because: "the LA Studio Comp has no Attack",
			},
			want: true,
		},
		{
			// The other silence: nothing anywhere acts on this word, so the
			// chain is not what is missing.
			name: "nothing acts on it at all",
			in:   result.Moved{Term: "short-decay"},
		},
		{
			name: "a word that moved a control",
			in:   result.Moved{Term: "mid-forward", Param: "Mid"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Unanswered())
		})
	}
}

// TestHolds covers a word the chain answers without a knob being turned.
func (s *MadePublicTestSuite) TestHolds() {
	tests := []struct {
		name string
		in   result.Moved
		want bool
	}{
		{
			// Nothing was turned and nothing is missing. A chain with no
			// reverb in it has no room on it, which is what dry asked for.
			name: "the chain is already what was asked for",
			in: result.Moved{
				Term:    "dry",
				Already: "this chain has no reverb, so it is already dry",
			},
			want: true,
		},
		{
			name: "a word the chain could not answer",
			in: result.Moved{
				Term:    "roomy",
				Because: "this chain holds no reverb",
			},
		},
		{
			name: "a word that moved a control",
			in:   result.Moved{Term: "mid-forward", Param: "Mid"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Holds())
		})
	}
}

func TestMadePublicTestSuite(t *testing.T) {
	suite.Run(t, new(MadePublicTestSuite))
}
