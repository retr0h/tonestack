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
	"math"
	"os"
	"path/filepath"
	"testing"

	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// DirPublicTestSuite covers measuring a directory of recordings.
type DirPublicTestSuite struct {
	suite.Suite

	root string
}

func (s *DirPublicTestSuite) SetupTest() {
	s.root = s.T().TempDir()
}

// put writes a signal to a path under the root, creating what it needs.
func (s *DirPublicTestSuite) put(
	at string,
	samples []float64,
) {
	full := filepath.Join(s.root, filepath.FromSlash(at))

	s.Require().NoError(os.MkdirAll(filepath.Dir(full), 0o750))

	f, err := os.Create(full) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	enc := wav.NewEncoder(f, rate, 16, 1, 1)

	scale := math.Pow(2, 15) - 1
	data := make([]int, 0, len(samples))

	for _, v := range samples {
		data = append(data, int(v*scale))
	}

	s.Require().NoError(enc.Write(&goaudio.IntBuffer{
		Format:         &goaudio.Format{NumChannels: 1, SampleRate: rate},
		Data:           data,
		SourceBitDepth: 16,
	}))
	s.Require().NoError(enc.Close())
	s.Require().NoError(f.Close())
}

// measureAll runs the walk over the root the way a caller would.
func (s *DirPublicTestSuite) measureAll() []audio.Named {
	got, err := audio.MeasureAll(os.DirFS(s.root), ".")
	s.Require().NoError(err)

	return got
}

// TestItFindsEveryRecording covers the walk reaching a whole tree.
func (s *DirPublicTestSuite) TestItFindsEveryRecording() {
	s.put("longview.wav", audio.Sine(110, 0.5, rate, 0.8))
	s.put("nested/brain-stew.wav", audio.Sine(120, 0.5, rate, 0.8))

	got := s.measureAll()

	s.Require().Len(got, 2)
}

// TestAStemIsNamedForItsTrack covers the separation tool's layout.
//
// Demucs writes one directory per track holding stems named for the
// instrument. Four rows reading "bass" would name nothing.
func (s *DirPublicTestSuite) TestAStemIsNamedForItsTrack() {
	s.put("htdemucs/longview/bass.wav", audio.Sine(110, 0.5, rate, 0.8))
	s.put("htdemucs/basket-case/bass.wav", audio.Sine(120, 0.5, rate, 0.8))

	got := s.measureAll()

	s.Require().Len(got, 2)
	s.Require().Equal("basket-case", got[0].Name)
	s.Require().Equal("longview", got[1].Name)
}

// TestTheOtherHalfOfASeparationIsIgnored covers what two-stem mode writes.
//
// Demucs writes bass.wav and no_bass.wav side by side. Measuring both would
// put two rows under one track name and build a corpus half made of the parts
// the bass was taken out of.
func (s *DirPublicTestSuite) TestTheOtherHalfOfASeparationIsIgnored() {
	s.put("htdemucs/longview/bass.wav", audio.Sine(110, 0.5, rate, 0.8))
	s.put("htdemucs/longview/no_bass.wav", audio.Sine(3000, 0.5, rate, 0.8))

	got := s.measureAll()

	s.Require().Len(got, 1, "the accompaniment is not a second record")
	s.Require().Equal("longview", got[0].Name)
	s.Require().Greater(got[0].Profile.Low, 0.95, "and what stayed is the bass")
}

// TestARecordingKeepsItsOwnName covers a file that is not a stem.
func (s *DirPublicTestSuite) TestARecordingKeepsItsOwnName() {
	s.put("takes/longview.wav", audio.Sine(110, 0.5, rate, 0.8))

	got := s.measureAll()

	s.Require().Len(got, 1)
	s.Require().Equal("longview", got[0].Name)
}

// TestTheOrderIsFixed covers two runs reading the same way.
func (s *DirPublicTestSuite) TestTheOrderIsFixed() {
	for _, name := range []string{"when-i-come-around", "basket-case", "longview"} {
		s.put(name+".wav", audio.Sine(110, 0.5, rate, 0.8))
	}

	got := s.measureAll()

	s.Require().Equal(
		[]string{"basket-case", "longview", "when-i-come-around"},
		[]string{got[0].Name, got[1].Name, got[2].Name},
	)
}

// TestItMeasuresWhatItFound covers the profiles being real.
func (s *DirPublicTestSuite) TestItMeasuresWhatItFound() {
	s.put("deep.wav", audio.Sine(80, 0.5, rate, 0.8))
	s.put("high.wav", audio.Sine(5000, 0.5, rate, 0.8))

	got := s.measureAll()

	s.Require().Greater(got[0].Profile.Low, 0.95, "the 80Hz one is low")
	s.Require().Greater(got[1].Profile.High, 0.95, "and the 5kHz one is not")
}

// TestAnythingThatIsNotAWavIsLeftAlone covers a directory with other files.
func (s *DirPublicTestSuite) TestAnythingThatIsNotAWavIsLeftAlone() {
	s.put("longview.wav", audio.Sine(110, 0.5, rate, 0.8))

	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, "SOURCES.md"), []byte("where these came from"), 0o600))

	got := s.measureAll()

	s.Require().Len(got, 1)
}

// TestAnEmptyDirectoryMeasuresAsNothing covers a caller pointed somewhere
// wrong, which is a report rather than an error.
func (s *DirPublicTestSuite) TestAnEmptyDirectoryMeasuresAsNothing() {
	s.Require().Empty(s.measureAll())
}

// TestAFileThatIsNotAudioIsNamed covers the error saying which one failed.
func (s *DirPublicTestSuite) TestAFileThatIsNotAudioIsNamed() {
	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, "broken.wav"), []byte("not a wav at all"), 0o600))

	_, err := audio.MeasureAll(os.DirFS(s.root), ".")

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "broken.wav")
}

// TestARootThatIsNotThere is a caller's mistake, reported rather than
// swallowed.
func (s *DirPublicTestSuite) TestARootThatIsNotThere() {
	_, err := audio.MeasureAll(os.DirFS(s.root), "no-such-directory")

	s.Require().Error(err)
}

func TestDirPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DirPublicTestSuite))
}
