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

// TestAWordSaysHowFarPastTheLineItSits covers the margin, which is what
// tells a word that will never flip from one that flips on the next player.
func (s *CorpusPublicTestSuite) TestAWordSaysHowFarPastTheLineItSits() {
	for _, player := range []string{"low-one", "low-two", "low-three"} {
		for _, track := range []string{"one", "two", "three"} {
			s.record(player, track, audio.Sine(80, 1, rate, 0.8))
		}
	}

	for _, track := range []string{"one", "two", "three"} {
		s.record("bright-one", track, audio.Sine(2000, 1, rate, 0.8))
	}

	for _, p := range s.corpus() {
		for _, t := range p.Terms {
			if t.Term == "bright" {
				s.Require().Positive(t.Margin,
					"a word is earned by clearing a line, so it clears it by something")
				s.Require().Greater(t.Mine, t.Others)

				return
			}
		}
	}

	s.Require().Fail("nobody earned bright")
}

// manifest writes one player's manifest, naming the records given.
func (s *CorpusPublicTestSuite) manifest(
	player string,
	tracks ...string,
) {
	body := "artist: " + player + "\ntracks:\n"
	for _, t := range tracks {
		body += "  - track: " + t +
			"\n    url: https://open.spotify.com/track/abc\n    year: 1994\n"
	}

	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, player, "corpus.yaml"), []byte(body), 0o600))
}

// TestOnlyWhatTheManifestNamesIsMeasured covers the manifest deciding what a
// player's figures are made of, rather than whatever is on disk.
//
// Records get replaced, and the workflow says to keep the audio of the ones
// taken out because separating it again costs minutes. Six were retired that
// way and every one of them was still in the figures until this: the counts
// read six where the manifest said three.
func (s *CorpusPublicTestSuite) TestOnlyWhatTheManifestNamesIsMeasured() {
	for _, track := range []string{"kept-one", "kept-two", "retired"} {
		s.record("mike-dirnt", track, audio.Sine(110, 1, rate, 0.8))
	}

	s.manifest("mike-dirnt", "kept-one", "kept-two")

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal(2, got[0].Records, "the retired record is still on disk")
}

// TestAPlayerWithNoManifestIsMeasuredAsFound covers a directory somebody is
// still assembling, where the manifest has not been written yet.
func (s *CorpusPublicTestSuite) TestAPlayerWithNoManifestIsMeasuredAsFound() {
	for _, track := range []string{"one", "two", "three"} {
		s.record("nobody", track, audio.Sine(110, 1, rate, 0.8))
	}

	got := s.corpus()

	s.Require().Len(got, 1)
	s.Require().Equal(3, got[0].Records)
}

// TestAManifestThatWillNotRead covers a corpus somebody broke, which stops
// the reading rather than silently measuring the tree instead.
func (s *CorpusPublicTestSuite) TestAManifestThatWillNotRead() {
	s.record("mike-dirnt", "one", audio.Sine(110, 1, rate, 0.8))
	s.Require().NoError(os.WriteFile(
		filepath.Join(s.root, "mike-dirnt", "corpus.yaml"),
		[]byte("tracks:\n  - track: one\n    note: a note: with a colon\n"), 0o600))

	_, err := audio.Corpus(os.DirFS(s.root), ".")

	s.Require().Error(err)
}

// TestAManifestThatCannotBeOpened covers a manifest that is there and will
// not open.
//
// Distinct from one that is absent, which measures the tree as found: a
// manifest nobody can read is a statement of what to measure that nobody can
// read, and measuring the tree instead would quietly use records somebody
// took out.
//
// A symlink to itself rather than a file with its permissions removed. Both
// fail to open; only one fails for every user, and a test that skips itself
// for root is a test that does not run where it matters.
func (s *CorpusPublicTestSuite) TestAManifestThatCannotBeOpened() {
	s.record("mike-dirnt", "one", audio.Sine(110, 1, rate, 0.8))

	at := filepath.Join(s.root, "mike-dirnt", "corpus.yaml")
	s.Require().NoError(os.Symlink("corpus.yaml", at))

	_, err := audio.Corpus(os.DirFS(s.root), ".")

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "mike-dirnt")
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
