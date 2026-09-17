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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.yaml.in/yaml/v3"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// EvidencePublicTestSuite covers measurements written as rig evidence.
type EvidencePublicTestSuite struct {
	suite.Suite
}

// records is two recordings with something in every measure.
func (s *EvidencePublicTestSuite) records() []audio.Named {
	return []audio.Named{
		{Name: "longview", Profile: audio.Profile{
			Low: 0.91, Mid: 0.08, High: 0.01,
			Centroid:     175,
			Transient:    audio.Reading{Value: 0.74, Known: true},
			Decay:        audio.Reading{Value: 0.82, Known: true},
			DynamicRange: 7.8,
			Harmonics:    audio.Spread{Mid: 0.35},
			EvenOdd:      audio.Spread{Mid: 0.9},
		}},
		{Name: "basket-case", Profile: audio.Profile{
			Low: 0.97, Centroid: 151,
			Transient:    audio.Reading{Value: 0.79, Known: true},
			Decay:        audio.Reading{Value: 1.72, Known: true},
			DynamicRange: 5.3,
			Harmonics:    audio.Spread{Mid: 0.23},
		}},
	}
}

// render writes evidence and hands back what was written.
func (s *EvidencePublicTestSuite) render(
	of []audio.Named,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Evidence(&buf, of))

	return buf.String()
}

// TestItParsesBackAsEvidence is the contract this output has to meet.
//
// The point of writing it is that somebody pastes it into a rig. Text that
// looks right and does not load is worse than no output, so this reads it
// back through the same type a rig is built from.
func (s *EvidencePublicTestSuite) TestItParsesBackAsEvidence() {
	var got []rig.Evidence

	s.Require().NoError(yaml.Unmarshal([]byte(s.render(s.records())), &got))

	s.Require().Len(got, 2)

	for _, e := range got {
		s.Require().Equal(rig.EvidenceKind("audio"), e.Kind)
		s.Require().NotNil(e.Measured)
		s.Require().NotNil(e.Caveat)
		s.Require().Len(*e.Measured, len(audio.MeasuredKeys()))
	}
}

// TestTheFiguresSurviveTheTrip covers the numbers arriving unchanged.
func (s *EvidencePublicTestSuite) TestTheFiguresSurviveTheTrip() {
	var got []rig.Evidence

	s.Require().NoError(yaml.Unmarshal([]byte(s.render(s.records())), &got))

	m := *got[0].Measured

	s.Require().InDelta(0.91, m[audio.KeyLow], 1e-9)
	s.Require().InDelta(7.8, m[audio.KeyDynamics], 1e-9)
	s.Require().InDelta(0.35, m[audio.KeyHarmonics], 1e-9)

	// A figure landing on a whole number is written untagged, so this is also
	// what holds the decoder to reading 175 as the number 175.
	s.Require().InDelta(175, m[audio.KeyCentroid], 1e-9)
}

// TestATimestampStaysAString is the one value a rig cannot afford to guess at.
//
// Rigs are written by a YAML 1.2 encoder and loaded by a 1.1 decoder, and in
// 1.1 colon-separated digits are sexagesimal: a bare `at: 1:42` comes back as
// the number 102. A single timestamp is the case that exposes it, because a
// range carries a trailing `-1:45` that keeps it a string by accident.
func (s *EvidencePublicTestSuite) TestATimestampStaysAString() {
	var buf bytes.Buffer

	s.Require().NoError(cli.Evidence(&buf, []audio.Named{
		{Name: "longview", Source: audio.Record{
			URL: "https://example.com/a", At: "1:42",
		}},
	}))

	s.Require().Contains(buf.String(), `at: "1:42"`)

	var got []rig.Evidence

	s.Require().NoError(yaml.Unmarshal(buf.Bytes(), &got))
	s.Require().NotNil(got[0].At)
	s.Require().Equal("1:42", *got[0].At)
}

// TestWhatTheManifestKnowsReachesTheEntry covers the link arriving.
func (s *EvidencePublicTestSuite) TestWhatTheManifestKnowsReachesTheEntry() {
	var buf bytes.Buffer

	s.Require().NoError(cli.Evidence(&buf, []audio.Named{
		{Name: "longview", Source: audio.Record{
			URL:  "https://open.spotify.com/track/abc",
			At:   "1:20-1:45",
			Note: "the bass carries the verse alone",
		}},
	}))

	var got []rig.Evidence

	s.Require().NoError(yaml.Unmarshal(buf.Bytes(), &got))

	s.Require().Equal("https://open.spotify.com/track/abc", *got[0].URL)
	s.Require().Equal("1:20-1:45", *got[0].At)
	s.Require().Contains(*got[0].Note, "the bass carries the verse alone")
}

// TestWithoutAManifestThereIsNoLink covers the entry a corpus with no
// manifest produces.
//
// Worse evidence rather than broken evidence, which is why it is allowed.
func (s *EvidencePublicTestSuite) TestWithoutAManifestThereIsNoLink() {
	var got []rig.Evidence

	s.Require().NoError(yaml.Unmarshal([]byte(s.render(s.records())), &got))

	s.Require().Nil(got[0].URL)
	s.Require().Nil(got[0].At)
	s.Require().NotNil(got[0].Measured, "the measurement is still there")
}

// TestEachRecordingIsItsOwnEntry covers evidence staying per source.
//
// A url makes a claim checkable, and one entry averaging four records is the
// one thing nobody could check.
func (s *EvidencePublicTestSuite) TestEachRecordingIsItsOwnEntry() {
	got := s.render(s.records())

	s.Require().Contains(got, "longview")
	s.Require().Contains(got, "basket-case")
}

// TestItSaysToAddALink covers the output naming what it is missing.
func (s *EvidencePublicTestSuite) TestItSaysToAddALink() {
	got := s.render(s.records())

	s.Require().Contains(got, "url:")
	s.Require().Contains(got, "evidence:")
}

// TestTheKeyOrderIsFixed covers two runs writing the same text.
func (s *EvidencePublicTestSuite) TestTheKeyOrderIsFixed() {
	s.Require().Equal(s.render(s.records()), s.render(s.records()))

	got := s.render(s.records())

	at := -1
	for _, key := range audio.MeasuredKeys() {
		next := strings.Index(got, "\n    "+key+":")
		s.Require().Greater(next, at, "%s is written out of order", key)

		at = next
	}
}

// TestNothingToWrite covers a caller with no recordings.
//
// An empty list rather than a failure. Nothing measured is a fact about the
// directory, and the caller has already been told which directory it was.
func (s *EvidencePublicTestSuite) TestNothingToWrite() {
	got := s.render(nil)

	s.Require().NotContains(got, "kind: audio")

	var back []rig.Evidence

	s.Require().NoError(yaml.Unmarshal([]byte(got), &back))
	s.Require().Empty(back)
}

// TestAWriteThatFailsIsReported covers the output going somewhere that stops
// accepting it.
//
// Every point it can fail, rather than the first. The encoder writes some of
// the document and then flushes the rest when it is closed, so a failure
// arriving late is a different path from one arriving at the start, and only
// walking the whole range reaches both.
func (s *EvidencePublicTestSuite) TestAWriteThatFailsIsReported() {
	writes := &counting{}
	s.Require().NoError(cli.Evidence(writes, s.records()))

	for ok := range writes.n {
		s.Run(fmt.Sprintf("after %d writes", ok), func() {
			err := cli.Evidence(&stops{ok: ok}, s.records())

			s.Require().Error(err)
			s.Require().Contains(err.Error(), "writing evidence")
		})
	}
}

func TestEvidencePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EvidencePublicTestSuite))
}
