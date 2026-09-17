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
package cli_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// TermsPublicTestSuite covers the earned words written as evidence to paste.
type TermsPublicTestSuite struct {
	suite.Suite
}

// render writes a set of players and hands back what was written.
func (s *TermsPublicTestSuite) render(
	of []audio.Player,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.PlayerTerms(&buf, of))

	return buf.String()
}

// TestBothSidesOfTheComparisonAreWritten is the contract: a word without the
// figures that earned it is an assertion, and the compiler cannot size a move
// from one side alone.
func (s *TermsPublicTestSuite) TestBothSidesOfTheComparisonAreWritten() {
	got := s.render([]audio.Player{
		{
			ID:      "flea",
			Records: 3,
			Terms: []audio.Derived{{
				Term: "clean", Key: audio.KeyHarmonics,
				Why:  "share of energy above the fundamental",
				Mine: 0.12, Others: 0.24, Of: 5,
			}},
		},
	})

	s.Require().Contains(got, "flea:")
	s.Require().Contains(got, "term: clean")
	s.Require().Contains(got, "kind: audio")
	s.Require().Contains(got, "measured: {harmonics: 0.12}")
	s.Require().Contains(got, "against: {harmonics: 0.24}")
	s.Require().Contains(got, "against 4 other players")
	s.Require().Contains(got, "3 recordings")
	s.Require().Contains(got, "measures the record rather than the player")
}

// TestAPlayerWhoEarnedNothingIsNotWritten covers the ordinary case not
// filling the output with empty blocks.
func (s *TermsPublicTestSuite) TestAPlayerWhoEarnedNothingIsNotWritten() {
	got := s.render([]audio.Player{
		{ID: "les-claypool", Records: 3},
		{ID: "pino-palladino", Records: 3, Terms: []audio.Derived{{
			Term: "dark", Key: audio.KeyCentroid, Why: "centre of gravity",
			Mine: 96, Others: 170, Of: 5,
		}}},
	})

	s.Require().NotContains(got, "les-claypool")
	s.Require().Contains(got, "pino-palladino")
	s.Require().Contains(got, "measured: {centroid: 96}")
}

// TestNobodyEarnedAnything covers an empty answer saying what it means.
//
// An empty document reads as a tool that failed. Mixed evidence is the
// ordinary outcome and deserves a sentence.
func (s *TermsPublicTestSuite) TestNobodyEarnedAnything() {
	got := s.render([]audio.Player{{ID: "les-claypool", Records: 3}})

	s.Require().Contains(got, "no player's records earned a word")
	s.Require().NotContains(got, "les-claypool:")
}

// TestWhatToDoWithItIsSaidOnce covers the header a person needs.
func (s *TermsPublicTestSuite) TestWhatToDoWithItIsSaidOnce() {
	got := s.render([]audio.Player{
		{ID: "flea", Records: 3, Terms: []audio.Derived{{
			Term: "clean", Key: audio.KeyHarmonics, Mine: 0.12, Others: 0.24, Of: 5,
		}}},
	})

	s.Require().Contains(got, "Paste under the rig of the player it names")
	s.Require().Contains(got, "an argument, not a verdict")
}

// earned is a player with one word to write.
func (s *TermsPublicTestSuite) earned() []audio.Player {
	return []audio.Player{{
		ID: "flea", Records: 3, Terms: []audio.Derived{{
			Term: "clean", Key: audio.KeyHarmonics, Mine: 0.12, Others: 0.24, Of: 5,
		}},
	}}
}

// TestAWriteThatFailsIsReported covers the output going somewhere that stops
// accepting it.
//
// Every point it can fail, rather than the first: the encoder writes some of
// the document and flushes the rest when it is closed, so a late failure is a
// different path from an early one.
func (s *TermsPublicTestSuite) TestAWriteThatFailsIsReported() {
	writes := &counting{}
	s.Require().NoError(cli.PlayerTerms(writes, s.earned()))

	for ok := range writes.n {
		s.Run(fmt.Sprintf("after %d writes", ok), func() {
			err := cli.PlayerTerms(&stops{ok: ok}, s.earned())

			s.Require().Error(err)
			s.Require().Contains(err.Error(), "writing terms")
		})
	}
}

// TestAWriteThatFailsWithNothingEarnedIsReported covers the other output,
// which is one line and not a document.
func (s *TermsPublicTestSuite) TestAWriteThatFailsWithNothingEarnedIsReported() {
	err := cli.PlayerTerms(&stops{}, []audio.Player{{ID: "les-claypool", Records: 3}})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing terms")
}

func TestTermsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TermsPublicTestSuite))
}
