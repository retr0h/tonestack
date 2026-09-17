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
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// ReadPublicTestSuite covers getting samples off disk.
//
// Checked by writing a signal whose measurements are already known, reading it
// back, and measuring it again. A decoder that scales wrongly, drops a channel
// or misreads a bit depth changes those numbers, which is the whole reason
// this sits under a measurement rather than beside it.
type ReadPublicTestSuite struct {
	suite.Suite
}

// write puts samples in a WAV file and hands back the path.
func (s *ReadPublicTestSuite) write(
	samples []float64,
	rate int,
	channels int,
	depth int,
) string {
	path := filepath.Join(s.T().TempDir(), "signal.wav")

	f, err := os.Create(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	enc := wav.NewEncoder(f, rate, depth, channels, 1)

	full := math.Pow(2, float64(depth-1)) - 1
	data := make([]int, 0, len(samples)*channels)

	for _, v := range samples {
		// The same value in every channel, so averaging them returns it.
		for range channels {
			data = append(data, int(v*full))
		}
	}

	s.Require().NoError(enc.Write(&goaudio.IntBuffer{
		Format:         &goaudio.Format{NumChannels: channels, SampleRate: rate},
		Data:           data,
		SourceBitDepth: depth,
	}))
	s.Require().NoError(enc.Close())
	s.Require().NoError(f.Close())

	return path
}

// read opens a written file the way a caller would.
func (s *ReadPublicTestSuite) read(
	path string,
) ([]float64, int) {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	got, rate, err := audio.Read(f)
	s.Require().NoError(err)

	return got, rate
}

// TestASineSurvivesTheTrip is the whole contract in one case.
func (s *ReadPublicTestSuite) TestASineSurvivesTheTrip() {
	want := audio.Sine(440, 0.5, rate, 0.8)

	got, gotRate := s.read(s.write(want, rate, 1, 16))

	s.Require().Equal(rate, gotRate)
	s.Require().Len(got, len(want))

	// Within a 16-bit step. Writing a whole number and reading it back cannot
	// be exact, and a measurement does not need it to be.
	for i := range want {
		s.Require().InDelta(want[i], got[i], 0.001, "sample %d", i)
	}
}

// TestTheMeasurementsAreTheSameOffDisk is why the decoder matters.
func (s *ReadPublicTestSuite) TestTheMeasurementsAreTheSameOffDisk() {
	want := audio.Square(200, 1.0, rate, 0.5)
	direct := audio.Measure(want, rate)

	got, gotRate := s.read(s.write(want, rate, 1, 16))
	fromDisk := audio.Measure(got, gotRate)

	s.Require().InDelta(direct.Low, fromDisk.Low, 0.01)
	s.Require().InDelta(direct.Mid, fromDisk.Mid, 0.01)
	s.Require().InDelta(direct.Centroid, fromDisk.Centroid, 5)
	s.Require().InDelta(direct.Harmonics.Mid, fromDisk.Harmonics.Mid, 0.02)
}

// TestStereoIsAveragedToOne covers several channels becoming one.
func (s *ReadPublicTestSuite) TestStereoIsAveragedToOne() {
	want := audio.Sine(220, 0.5, rate, 0.7)

	got, _ := s.read(s.write(want, rate, 2, 16))

	s.Require().Len(got, len(want), "two channels in, one out")

	for i := range want {
		s.Require().InDelta(want[i], got[i], 0.001)
	}
}

// TestBitDepthDoesNotChangeTheLevel is the scaling that is easy to get wrong.
//
// The same take at 16 and 24 bits is the same sound. A decoder that forgot to
// divide by the width would report one as hundreds of times louder.
func (s *ReadPublicTestSuite) TestBitDepthDoesNotChangeTheLevel() {
	want := audio.Sine(300, 0.5, rate, 0.6)

	shallow, _ := s.read(s.write(want, rate, 1, 16))
	deep, _ := s.read(s.write(want, rate, 1, 24))

	s.Require().InDelta(
		audio.Loudness(shallow), audio.Loudness(deep), 0.001,
		"the same signal at two widths is the same loudness")
}

// TestSampleRateIsReported covers the rate coming back with the samples.
func (s *ReadPublicTestSuite) TestSampleRateIsReported() {
	for _, r := range []int{22050, 44100, 48000} {
		s.Run("", func() {
			_, got := s.read(s.write(audio.Sine(100, 0.2, r, 0.5), r, 1, 16))

			s.Require().Equal(r, got)
		})
	}
}

// TestSomethingThatIsNotAudio is refused rather than measured.
func (s *ReadPublicTestSuite) TestSomethingThatIsNotAudio() {
	got, rate, err := audio.Read(bytes.NewReader([]byte("this is not a wav")))

	s.Require().Nil(got)
	s.Require().Zero(rate)
	s.Require().ErrorIs(err, audio.ErrNotAudio)
}

// TestNothingAtAll is refused for the same reason.
func (s *ReadPublicTestSuite) TestNothingAtAll() {
	_, _, err := audio.Read(bytes.NewReader(nil))

	s.Require().ErrorIs(err, audio.ErrNotAudio)
	s.Require().Contains(err.Error(), "not readable audio")
}

// TestAWavHoldingNoSamples reads as nothing rather than failing.
func (s *ReadPublicTestSuite) TestAWavHoldingNoSamples() {
	got, gotRate, err := func() ([]float64, int, error) {
		f, err := os.Open(s.write(nil, rate, 1, 16)) //nolint:gosec // this test's own path
		s.Require().NoError(err)

		defer func() { s.Require().NoError(f.Close()) }()

		return audio.Read(f)
	}()

	s.Require().NoError(err)
	s.Require().Empty(got)
	s.Require().Equal(rate, gotRate)
}

func TestReadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ReadPublicTestSuite))
}
