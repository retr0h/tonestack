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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// ManifestPublicTestSuite covers the record of what a corpus was.
type ManifestPublicTestSuite struct {
	suite.Suite
}

// read parses a manifest the way a caller would.
func (s *ManifestPublicTestSuite) read(
	doc string,
) audio.Manifest {
	got, err := audio.ReadManifest(strings.NewReader(doc))
	s.Require().NoError(err)

	return got
}

// full is a manifest with something in every field.
const full = `
artist: Mike Dirnt
tracks:
  - track: longview
    url: https://open.spotify.com/track/abc
    at: "1:20-1:45"
    note: the bass carries the verse alone
  - track: basket-case
    url: https://open.spotify.com/track/def
`

// TestItReadsWhatWasMeasured covers the whole shape.
func (s *ManifestPublicTestSuite) TestItReadsWhatWasMeasured() {
	got := s.read(full)

	s.Require().Equal("Mike Dirnt", got.Artist)
	s.Require().Len(got.Tracks, 2)

	s.Require().Equal("longview", got.Tracks[0].Track)
	s.Require().Equal("https://open.spotify.com/track/abc", got.Tracks[0].URL)
	s.Require().Equal("1:20-1:45", got.Tracks[0].At)
	s.Require().Equal("the bass carries the verse alone", got.Tracks[0].Note)
}

// TestItNamesNoFiles is the point of a manifest rather than a directory
// listing.
//
// The audio is somebody else's and cannot be committed. What can is the
// record of which songs were measured, and that record is worth nothing if it
// only works on the machine that holds them.
func (s *ManifestPublicTestSuite) TestItNamesNoFiles() {
	for _, rec := range s.read(full).Tracks {
		s.Require().NotContains(rec.URL, "/Users/")
		s.Require().NotContains(rec.URL, ".wav")
	}
}

// TestATypoStops covers a field nobody meant to write.
//
// `track` and `tracks` are one letter apart, and a manifest that silently
// measures nothing is worse than one that refuses.
func (s *ManifestPublicTestSuite) TestATypoStops() {
	_, err := audio.ReadManifest(strings.NewReader("artist: x\ntrack:\n  - track: y\n"))

	s.Require().Error(err)
}

// TestABadTimestampIsCaughtHere covers the check happening where it can be
// acted on.
//
// Left until build time this surfaces against `chain[0].evidence[1].at`,
// which says nothing about which song was wrong.
func (s *ManifestPublicTestSuite) TestABadTimestampIsCaughtHere() {
	_, err := audio.ReadManifest(strings.NewReader(
		"tracks:\n  - track: longview\n    at: \"about a minute in\"\n"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "longview")
	s.Require().Contains(err.Error(), "timestamp")
}

// TestABadLinkIsCaughtHere covers the other thing a rig will refuse.
func (s *ManifestPublicTestSuite) TestABadLinkIsCaughtHere() {
	_, err := audio.ReadManifest(strings.NewReader(
		"tracks:\n  - track: longview\n    url: spotify:track:abc\n"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "longview")
}

// TestAnEntryWithNoTrackStops covers a record naming nothing.
func (s *ManifestPublicTestSuite) TestAnEntryWithNoTrackStops() {
	_, err := audio.ReadManifest(strings.NewReader(
		"tracks:\n  - url: https://example.com/a\n"))

	s.Require().Error(err)
}

// TestItIsNotYaml covers a file that is not a manifest at all.
func (s *ManifestPublicTestSuite) TestItIsNotYaml() {
	_, err := audio.ReadManifest(strings.NewReader("\tnot: [a manifest"))

	s.Require().Error(err)
}

// TestJoinAttachesTheSource covers a measurement gaining its link.
func (s *ManifestPublicTestSuite) TestJoinAttachesTheSource() {
	got := s.read(full).Join([]audio.Named{
		{Name: "longview"},
		{Name: "brain-stew"},
	})

	s.Require().Equal("https://open.spotify.com/track/abc", got[0].Source.URL)
	s.Require().Equal("1:20-1:45", got[0].Source.At)

	s.Require().Empty(got[1].Source.URL,
		"a measurement the manifest does not mention is still a measurement")
}

// TestJoinKeepsEverything covers nothing being dropped for want of a link.
func (s *ManifestPublicTestSuite) TestJoinKeepsEverything() {
	in := []audio.Named{{Name: "longview"}, {Name: "nothing-named-this"}}

	s.Require().Len(s.read(full).Join(in), len(in))
}

// TestJoinIgnoresCase covers a manifest written by a person.
func (s *ManifestPublicTestSuite) TestJoinIgnoresCase() {
	got := s.read("tracks:\n  - track: LongView\n    url: https://example.com/a\n").
		Join([]audio.Named{{Name: "longview"}})

	s.Require().Equal("https://example.com/a", got[0].Source.URL)
}

// TestUnmatchedReportsBothDirections covers the two mistakes worth telling.
func (s *ManifestPublicTestSuite) TestUnmatchedReportsBothDirections() {
	missing, unnamed := s.read(full).Unmatched([]audio.Named{
		{Name: "longview"},
		{Name: "brain-stew"},
	})

	s.Require().Equal([]string{"basket-case"}, missing,
		"named in the manifest, nothing measured")
	s.Require().Equal([]string{"brain-stew"}, unnamed,
		"measured, and its evidence will go out with no link")
}

// TestUnmatchedIsQuietWhenTheyAgree covers the ordinary case.
func (s *ManifestPublicTestSuite) TestUnmatchedIsQuietWhenTheyAgree() {
	missing, unnamed := s.read(full).Unmatched([]audio.Named{
		{Name: "longview"},
		{Name: "basket-case"},
	})

	s.Require().Empty(missing)
	s.Require().Empty(unnamed)
}

func TestManifestPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ManifestPublicTestSuite))
}
