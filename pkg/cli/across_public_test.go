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

// AcrossPublicTestSuite covers the corpus table.
type AcrossPublicTestSuite struct {
	suite.Suite
}

// render draws a gathered measurement and hands back what was written.
func (s *AcrossPublicTestSuite) render(
	a audio.Across,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Across(&buf, a))

	return buf.String()
}

// full is a gathered measurement with something in every field.
func (s *AcrossPublicTestSuite) full() audio.Across {
	return audio.Across{
		Tracks:   4,
		Low:      audio.Spread{Low: 0.91, Mid: 0.94, High: 0.97},
		Mid:      audio.Spread{Low: 0.02, Mid: 0.05, High: 0.08},
		High:     audio.Spread{Low: 0.01, Mid: 0.01, High: 0.02},
		Centroid: audio.Spread{Low: 135, Mid: 150, High: 191},
		Transient: audio.Ranged{
			Spread: audio.Spread{Low: 0.68, Mid: 0.72, High: 0.77}, From: 4,
		},
		Decay: audio.Ranged{
			Spread: audio.Spread{Low: 0.15, Mid: 0.82, High: 1.72}, From: 4,
		},
		DynamicRange: audio.Spread{Low: 5.5, Mid: 6.1, High: 7.6},
		Harmonics:    audio.Spread{Low: 0.18, Mid: 0.21, High: 0.35},
		EvenOdd:      audio.Spread{Low: 0.1, Mid: 0.4, High: 0.6},
	}
}

// TestEveryMeasureReachesThePage is the whole contract.
func (s *AcrossPublicTestSuite) TestEveryMeasureReachesThePage() {
	got := s.render(s.full())

	s.Require().Contains(got, "4 recordings")

	s.Require().Contains(got, "150 Hz")
	s.Require().Contains(got, "135 Hz–191 Hz", "and the width around it")

	s.Require().Contains(got, "0.72")
	s.Require().Contains(got, "0.82 s")
	s.Require().Contains(got, "6.1 dB")
	s.Require().Contains(got, "21%")
}

// TestEveryMeasureIsNamed covers the rows being readable.
func (s *AcrossPublicTestSuite) TestEveryMeasureIsNamed() {
	got := s.render(s.full())

	for _, name := range []string{
		"low", "mid", "high", "centroid", "transient", "decay", "dynamics",
		"harmonics", "lean",
	} {
		s.Require().Contains(got, name)
	}
}

// TestItReachesNoVerdict is the line this renderer does not cross either.
func (s *AcrossPublicTestSuite) TestItReachesNoVerdict() {
	got := s.render(s.full())

	for _, verdict := range []string{
		"warm", "bright", "dark", "punchy", "percussive", "muddy", "harsh",
	} {
		s.Require().NotContains(got, verdict)
	}
}

// TestOneRecordingSaysSo covers the count reading as English.
//
// It is load-bearing rather than decoration: with few records the ends of
// every width are the extreme records.
func (s *AcrossPublicTestSuite) TestOneRecordingSaysSo() {
	a := s.full()
	a.Tracks = 1

	s.Require().Contains(s.render(a), "1 recording")
}

// TestNothingMeasured covers a corpus with no recordings in it.
func (s *AcrossPublicTestSuite) TestNothingMeasured() {
	got := s.render(audio.Across{})

	s.Require().Contains(got, "0 recordings")
}

// TestAMeasureOnlySomeRecordsAnsweredSaysHowMany covers the partial case.
//
// A middle taken over two records of four is still a measurement of those
// two. Reading it beside a middle taken over all four, with nothing marking
// which is which, is how a corpus quietly becomes a smaller one.
func (s *AcrossPublicTestSuite) TestAMeasureOnlySomeRecordsAnsweredSaysHowMany() {
	a := s.full()
	a.Decay = audio.Ranged{
		Spread: audio.Spread{Low: 0.80, Mid: 0.90, High: 1.00}, From: 2,
	}

	s.Require().Contains(s.render(a), "from 2 of 4")
}

// TestAMeasureNoRecordAnsweredSaysSo covers none of them having it.
func (s *AcrossPublicTestSuite) TestAMeasureNoRecordAnsweredSaysSo() {
	a := s.full()
	a.Decay = audio.Ranged{}

	got := s.render(a)

	s.Require().Contains(got, "no record answered this")
	s.Require().NotContains(got, "0.00 s", "and no figure standing in for one")
}

// TestEachRecordGetsARow covers the per-record table.
func (s *AcrossPublicTestSuite) TestEachRecordGetsARow() {
	var buf bytes.Buffer

	s.Require().NoError(cli.Tracks(&buf, []audio.Named{
		{Name: "basket-case", Profile: audio.Profile{
			Low: 0.97, Centroid: 148,
			Transient:    audio.Reading{Value: 0.77, Known: true},
			Decay:        audio.Reading{Value: 1.72, Known: true},
			DynamicRange: 5.5, Harmonics: audio.Spread{Mid: 0.23},
		}},
		{Name: "longview", Profile: audio.Profile{
			Low: 0.94, Centroid: 152,
			Transient:    audio.Reading{Value: 0.71, Known: true},
			Decay:        audio.Reading{Value: 0.82, Known: true},
			DynamicRange: 7.6, Harmonics: audio.Spread{Mid: 0.35},
		}},
	}))

	got := buf.String()

	s.Require().Contains(got, "basket-case")
	s.Require().Contains(got, "longview")
	s.Require().Contains(got, "148 Hz")
	s.Require().Contains(got, "1.72 s")
	s.Require().Contains(got, "7.6 dB")
	s.Require().Contains(got, "35%")
	s.Require().Contains(got, "2 recordings")
}

// TestNoRecordsAtAll covers the table with nothing in it.
func (s *AcrossPublicTestSuite) TestNoRecordsAtAll() {
	var buf bytes.Buffer

	s.Require().NoError(cli.Tracks(&buf, nil))

	s.Require().Contains(buf.String(), "no recordings to measure")
}

func TestAcrossPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AcrossPublicTestSuite))
}
