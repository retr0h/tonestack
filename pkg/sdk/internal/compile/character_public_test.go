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

	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// CharacterPublicTestSuite covers the words a rig may use for how it sounds.
type CharacterPublicTestSuite struct {
	suite.Suite
}

// TestCharacterTerms covers the shipped vocabulary.
func (s *CharacterPublicTestSuite) TestCharacterTerms() {
	got := compile.CharacterTerms()

	s.Require().NotEmpty(got)
	s.Require().Contains(got, "mid-forward")
	s.Require().Contains(got, "audible-pick-attack")

	// Sorted, so a suggestion reads the same way twice.
	for i := 1; i < len(got); i++ {
		s.Require().Less(got[i-1], got[i])
	}
}

// TestCheckCharacter covers reporting the words nothing defines.
func (s *CharacterPublicTestSuite) TestCheckCharacter() {
	tests := []struct {
		name string
		in   []string
		want map[string][]string
	}{
		{name: "a rig saying nothing about how it sounds"},
		{
			name: "every word in the vocabulary",
			in:   []string{"mid-forward", "short-decay"},
		},
		{
			// The spelling every rig in this repository used before there
			// was a vocabulary, which is the case the suggestion is for.
			name: "a sentence where a term belongs",
			in:   []string{"pick attack audible"},
			want: map[string][]string{
				"pick attack audible": {"audible-pick-attack"},
			},
		},
		{
			// Two claims in one line, and the closest single term is all
			// that is offered. Naming both would read as alternatives when
			// what the writer wants is to say two things.
			name: "two claims welded into one line",
			in:   []string{"tight low end, short decay"},
			want: map[string][]string{
				"tight low end, short decay": {"tight-low-end"},
			},
		},
		{
			name: "a word sharing nothing with any of them",
			in:   []string{"zzz"},
			want: map[string][]string{"zzz": nil},
		},
		{
			// A term made of punctuation has no words to compare, so there
			// is nothing to be close to.
			name: "a term that is not a word at all",
			in:   []string{"---"},
			want: map[string][]string{"---": nil},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var spec rig.Spec

			if tt.in != nil {
				terms := make([]rig.CharacterTerm, 0, len(tt.in))
				for _, t := range tt.in {
					terms = append(terms, rig.CharacterTerm{Term: t})
				}

				spec.Character = &terms
			}

			got := compile.CheckCharacter(spec)

			s.Require().Len(got, len(tt.want))

			for _, u := range got {
				near, ok := tt.want[u.Term]
				s.Require().True(ok, "unexpected term %q", u.Term)
				s.Require().Subset(u.Near, near)
			}
		})
	}
}

func TestCharacterPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CharacterPublicTestSuite))
}
