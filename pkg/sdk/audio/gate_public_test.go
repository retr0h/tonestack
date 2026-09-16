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

package audio_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// GatePublicTestSuite covers telling the playing from the gaps between it.
//
// A rest is not quiet playing. Measuring one as though it were reports the
// silence between the notes, which is what a real bass part with rests in it
// did before this existed: 55dB of dynamic range on a player who sits between
// 5 and 8 everywhere else.
type GatePublicTestSuite struct {
	suite.Suite
}

// part is notes with rests between them, which is what playing looks like.
func (s *GatePublicTestSuite) part(
	notes int,
	note float64,
	rest float64,
) []float64 {
	out := make([]float64, 0, notes*int((note+rest)*float64(rate)))

	for range notes {
		out = append(out, audio.Plucked(110, note, rate, 0.8, 3)...)
		out = append(out, audio.Silence(rest, rate)...)
	}

	return out
}

// TestSilenceIsNotPlaying is the whole point.
func (s *GatePublicTestSuite) TestSilenceIsNotPlaying() {
	levels := audio.Frames(s.part(3, 0.5, 0.5), rate/50)
	held := audio.Playing(levels)

	var sounding, quiet int

	for _, on := range held {
		if on {
			sounding++
		} else {
			quiet++
		}
	}

	s.Require().Positive(sounding, "the notes are playing")
	s.Require().Positive(quiet, "the rests are not")
}

// TestAHeldToneIsAllPlaying has no gaps to find.
func (s *GatePublicTestSuite) TestAHeldToneIsAllPlaying() {
	levels := audio.Frames(audio.Sine(110, 1.0, rate, 0.8), rate/50)

	for i, on := range audio.Playing(levels) {
		s.Require().True(on, "frame %d", i)
	}
}

// TestSilenceIsNeverPlaying covers a signal with nothing in it.
func (s *GatePublicTestSuite) TestSilenceIsNeverPlaying() {
	levels := audio.Frames(audio.Silence(1.0, rate), rate/50)

	for i, on := range audio.Playing(levels) {
		s.Require().False(on, "frame %d", i)
	}
}

// TestNoFramesAtAll answers with nothing rather than dividing by it.
func (s *GatePublicTestSuite) TestNoFramesAtAll() {
	s.Require().Empty(audio.Playing(nil))
	s.Require().Empty(audio.Frames(nil, 100))
}

// TestTheFloorIsRelativeToTheRecording covers a quiet master and a loud one.
//
// The same performance mastered twice at different levels is the same
// playing, so the gate cannot be an absolute threshold.
func (s *GatePublicTestSuite) TestTheFloorIsRelativeToTheRecording() {
	loud := audio.Frames(s.part(3, 0.4, 0.4), rate/50)

	quiet := make([]float64, len(loud))
	for i, l := range loud {
		quiet[i] = l / 50
	}

	s.Require().Equal(audio.Playing(loud), audio.Playing(quiet))
}

// TestRestsNoLongerInflateTheDynamicRange is the defect this fixes.
//
// A part that rests half the time used to report the gap between its loudest
// note and the noise floor, which is not how dynamic the playing is.
func (s *GatePublicTestSuite) TestRestsNoLongerInflateTheDynamicRange() {
	steady := audio.Measure(s.part(6, 0.5, 0.05), rate)
	resting := audio.Measure(s.part(6, 0.5, 2.0), rate)

	// The notes are identical; only the silence between them differs.
	s.Require().InDelta(steady.DynamicRange, resting.DynamicRange, 3.0,
		"the rests between notes are not part of how dynamic the playing is")
}

// TestDecayStopsAtARest covers the other measurement silence broke.
//
// A note that is cut off by the player stopping did not decay over the rest
// that follows it, and timing one reports the gap rather than the ring.
func (s *GatePublicTestSuite) TestDecayStopsAtARest() {
	// One note that rings, then a long silence.
	got := audio.Measure(
		append(audio.Plucked(110, 1.0, rate, 0.8, 2), audio.Silence(5.0, rate)...),
		rate)

	s.Require().Less(got.Decay, 2.0,
		"the five seconds of silence are not part of the note")
	s.Require().Positive(got.Decay)
}

// TestQuietestIsWhereItSays guards the one number this all turns on.
func (s *GatePublicTestSuite) TestQuietestIsWhereItSays() {
	s.Require().InDelta(-40.0, audio.Quietest, 0.001,
		"measured against four real bass stems; changing it needs the same")
}

func TestGatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GatePublicTestSuite))
}
