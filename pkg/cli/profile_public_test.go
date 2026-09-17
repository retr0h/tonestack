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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// ProfilePublicTestSuite covers printing what a recording measures as.
//
// What matters is that every number reaches the page in the unit it is in, and
// that nothing here reads as a verdict. The numbers are the point; calling one
// of them warm is an argument this does not join.
type ProfilePublicTestSuite struct {
	suite.Suite
}

// render draws a profile and hands back what was written.
func (s *ProfilePublicTestSuite) render(
	p audio.Profile,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Profile(&buf, p))

	return buf.String()
}

// full is a profile with something in every field.
func (s *ProfilePublicTestSuite) full() audio.Profile {
	return audio.Profile{
		Seconds:      12.5,
		Rate:         44100,
		Low:          0.62,
		Mid:          0.31,
		High:         0.07,
		Centroid:     410,
		Transient:    0.81,
		Decay:        0.42,
		DynamicRange: 4.2,
		Harmonics:    audio.Spread{Low: 0.04, Mid: 0.18, High: 0.45},
		EvenOdd:      audio.Spread{Low: -0.8, Mid: -0.6, High: -0.2},
	}
}

// TestEveryNumberReachesThePage is the whole contract.
func (s *ProfilePublicTestSuite) TestEveryNumberReachesThePage() {
	got := s.render(s.full())

	s.Require().Contains(got, "12.5s")
	s.Require().Contains(got, "44100 Hz")

	s.Require().Contains(got, "62% low")
	s.Require().Contains(got, "31% mid")
	s.Require().Contains(got, "7% high")

	s.Require().Contains(got, "410 Hz")
	s.Require().Contains(got, "0.81")
	s.Require().Contains(got, "0.42 s")
	s.Require().Contains(got, "4.2 dB")
	s.Require().Contains(got, "18%")
	s.Require().Contains(got, "4–45%", "the middle window is not the whole answer")
}

// TestEveryMeasureIsNamed covers the rows being readable.
func (s *ProfilePublicTestSuite) TestEveryMeasureIsNamed() {
	got := s.render(s.full())

	for _, name := range []string{
		"energy", "centroid", "transient", "decay", "dynamics", "harmonics",
		"spread",
	} {
		s.Require().Contains(got, name)
	}
}

// TestItReachesNoVerdict is the line this renderer does not cross.
//
// The words a rig uses are judgements somebody made. A measurement is not one,
// and printing them beside each other would quietly turn one into the other.
func (s *ProfilePublicTestSuite) TestItReachesNoVerdict() {
	got := s.render(s.full())

	for _, verdict := range []string{
		"warm", "bright", "dark", "punchy", "percussive", "muddy", "harsh",
	} {
		s.Require().NotContains(got, verdict)
	}
}

// TestHarmonicLean says which harmonics carry more, and only that.
func (s *ProfilePublicTestSuite) TestHarmonicLean() {
	tests := []struct {
		name string
		lean float64
		want string
	}{
		{name: "odd, as a fuzz is", lean: -0.6, want: "leaning odd"},
		{name: "even, as a valve is", lean: 0.6, want: "leaning even"},
		{name: "neither in particular", lean: 0.0, want: "above the fundamental"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			p := s.full()
			p.EvenOdd = audio.Spread{Mid: tt.lean}

			s.Require().Contains(s.render(p), tt.want)
		})
	}
}

// TestAttackReads covers what the transient number is said to count.
func (s *ProfilePublicTestSuite) TestAttackReads() {
	tests := []struct {
		name      string
		transient float64
		want      string
	}{
		{name: "struck", transient: 0.95, want: "arrives in one step"},
		{name: "swelled", transient: 0.05, want: "climbs gradually"},
		{name: "in between", transient: 0.5, want: "largest single step"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			p := s.full()
			p.Transient = tt.transient

			s.Require().Contains(s.render(p), tt.want)
		})
	}
}

// TestCompressionReads covers the dynamics row naming what it compares.
func (s *ProfilePublicTestSuite) TestCompressionReads() {
	squashed := s.full()
	squashed.DynamicRange = 3

	open := s.full()
	open.DynamicRange = 18

	s.Require().Contains(s.render(squashed), "little room between them")
	s.Require().Contains(s.render(open), "loudest against typical")
}

// TestAnEmptyProfile draws rather than failing.
//
// Measuring silence is a real answer, and a renderer that could not print it
// would turn a fact into an error.
func (s *ProfilePublicTestSuite) TestAnEmptyProfile() {
	got := s.render(audio.Profile{})

	s.Require().Contains(got, "0 Hz")
	s.Require().Contains(got, "0%")
}

func TestProfilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProfilePublicTestSuite))
}
