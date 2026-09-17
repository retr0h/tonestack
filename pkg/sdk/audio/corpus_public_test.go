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

// CorpusPublicTestSuite covers measuring several players and comparing them.
type CorpusPublicTestSuite struct {
	suite.Suite

	root string
}

func (s *CorpusPublicTestSuite) SetupTest() {
	s.root = s.T().TempDir()
}

// record writes one of a player's recordings.
func (s *CorpusPublicTestSuite) record(
	player, track string,
	samples []float64,
) {
	full := filepath.Join(s.root, player, "stems", "htdemucs", track, "bass.wav")

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

// corpus reads the tree the way the command does.
func (s *CorpusPublicTestSuite) corpus() []audio.Player {
	got, err := audio.Corpus(os.DirFS(s.root), ".")
	s.Require().NoError(err)

	return got
}

// TestOnePlayerEarnsNothing covers why this exists.
//
// A word is earned by sitting clear of the other players. One player is clear
// of nobody, however far from the middle their records sit.
func (s *CorpusPublicTestSuite) TestOnePlayerEarnsNothing() {
	for _, track := range []string{"one", "two", "three"} {
		s.record("mike-dirnt", track, audio.Sine(800, 1, rate, 0.8))
	}

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal("mike-dirnt", got[0].ID)
	s.Require().Equal(3, got[0].Records)
	s.Require().Empty(got[0].Terms)
}

// TestAPlayerClearOfTheRestEarnsAWord covers the comparison doing its job.
//
// Three players low and one high, so the high one is the only one whose whole
// range sits above the others.
func (s *CorpusPublicTestSuite) TestAPlayerClearOfTheRestEarnsAWord() {
	for _, player := range []string{"low-one", "low-two", "low-three"} {
		for _, track := range []string{"one", "two", "three"} {
			s.record(player, track, audio.Sine(80, 1, rate, 0.8))
		}
	}

	for _, track := range []string{"one", "two", "three"} {
		s.record("bright-one", track, audio.Sine(2000, 1, rate, 0.8))
	}

	got := s.corpus()
	s.Require().Len(got, 4)

	terms := map[string][]string{}

	for _, p := range got {
		for _, t := range p.Terms {
			terms[p.ID] = append(terms[p.ID], t.Term)
		}
	}

	s.Require().Contains(terms["bright-one"], "bright")

	// The three at the bottom are identical, so which of them the quartile
	// lands on is decided by noise. What matters is that none of them is the
	// bright one.
	for _, player := range []string{"low-one", "low-two", "low-three"} {
		s.Require().NotContains(terms[player], "bright")
	}
}

// TestAPlayerIsComparedAgainstEverybodyElse covers a player not being
// compared against themselves.
func (s *CorpusPublicTestSuite) TestAPlayerIsComparedAgainstEverybodyElse() {
	for _, player := range []string{"one", "two"} {
		s.record(player, "track", audio.Sine(110, 1, rate, 0.8))
	}

	got := s.corpus()
	s.Require().Len(got, 2)

	for _, p := range got {
		for _, t := range p.Terms {
			s.Require().Equal(2, t.Of,
				"both players, counted once each")
		}
	}
}

// TestADirectoryWithNoRecordingsIsNotAPlayer covers a manifest waiting for
// audio somebody has not separated yet.
func (s *CorpusPublicTestSuite) TestADirectoryWithNoRecordingsIsNotAPlayer() {
	s.record("mike-dirnt", "longview", audio.Sine(110, 1, rate, 0.8))
	s.Require().NoError(os.MkdirAll(filepath.Join(s.root, "flea"), 0o750))
	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, "README.md"), []byte("not a player"), 0o600))

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal("mike-dirnt", got[0].ID)
}

// TestTheOrderIsFixed covers two runs reading the same way.
func (s *CorpusPublicTestSuite) TestTheOrderIsFixed() {
	for _, player := range []string{"pino-palladino", "flea", "mike-dirnt"} {
		s.record(player, "track", audio.Sine(110, 1, rate, 0.8))
	}

	got := s.corpus()

	s.Require().Len(got, 3)
	s.Require().Equal("flea", got[0].ID)
	s.Require().Equal("mike-dirnt", got[1].ID)
	s.Require().Equal("pino-palladino", got[2].ID)
}

// TestATreeThatIsNotThere covers the path being wrong.
func (s *CorpusPublicTestSuite) TestATreeThatIsNotThere() {
	_, err := audio.Corpus(os.DirFS(s.root), "nowhere")

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nowhere")
}

// TestARecordingThatIsNotOne covers a .wav that will not read.
func (s *CorpusPublicTestSuite) TestARecordingThatIsNotOne() {
	full := filepath.Join(s.root, "mike-dirnt", "broken.wav")
	s.Require().NoError(os.MkdirAll(filepath.Dir(full), 0o750))
	s.Require().NoError(os.WriteFile(full, []byte("not a wav"), 0o600))

	_, err := audio.Corpus(os.DirFS(s.root), ".")

	s.Require().Error(err)
}

func TestCorpusPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CorpusPublicTestSuite))
}
