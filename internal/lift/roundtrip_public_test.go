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

package lift_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/preset"
)

// corpusDir holds thousands of presets other people made. It is not committed
// — see docs/knowledge.md — so the sweep over it runs only where it exists,
// and the committed fixtures carry the same assertions everywhere else.
const corpusDir = "../../schemas/corpus"

type RoundTripPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *RoundTripPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

func (s *RoundTripPublicTestSuite) TestAPresetSurvivesBecomingARigAndBack() {
	for _, path := range s.fixtures() {
		s.Run(filepath.Base(path), func() {
			s.Require().Equal(s.canonical(s.read(path)), s.canonical(s.roundTrip(path)),
				"a preset read into a rig and written back must say the same thing")
		})
	}
}

func (s *RoundTripPublicTestSuite) TestALiftedRigRecordsTheExactModel() {
	// A gear name does not identify a model: 665 of them share 469 names, and
	// "Ampeg SVT" matches both channels. Without the identifier a rig rebuilds
	// into a different preset.
	doc, err := preset.Read(bytes.NewReader(s.read(s.fixtures()[0])))
	s.Require().NoError(err)

	spec, err := lift.Lift(doc, s.cat)
	s.Require().NoError(err)
	s.Require().NotEmpty(spec.Chain)

	for _, entry := range spec.Chain {
		s.Require().NotNil(entry.Models, "every block records what it was")
		s.Require().Contains(*entry.Models, s.cat.Device)
		s.Require().NotEmpty(entry.Gear, "and what a person would call it")
	}
}

func (s *RoundTripPublicTestSuite) TestTheWholeCorpusSurvivesIt() {
	if _, err := os.Stat(corpusDir); err != nil {
		s.T().Skip("no corpus here; the committed fixtures cover the same ground")
	}

	var (
		same, differ, skipped int
		failed                []string
	)

	for _, path := range s.corpus() {
		raw := s.read(path)

		doc, err := preset.Read(bytes.NewReader(raw))
		if err != nil || doc.Data.Device != s.cat.DeviceID {
			skipped++

			continue
		}

		// An empty slot is not a rig: it names no gear, and the schema says a
		// chain holds at least one thing.
		if c, err := doc.Spec(); err != nil || len(c.Blocks) == 0 {
			skipped++

			continue
		}

		if s.canonical(raw) == s.canonical(s.roundTrip(path)) {
			same++

			continue
		}

		differ++

		if len(failed) < 3 {
			failed = append(failed, path)
		}
	}

	s.T().Logf("round-tripped %d presets for this device, skipped %d others",
		same+differ, skipped)
	s.Require().Zero(differ,
		"every preset must survive becoming a rig and being written back; "+
			"first failures: %v", failed)
	s.Require().Positive(same)
}

// roundTrip reads a preset, lifts it to a rig, lowers it into a fresh read of
// the same document, and returns what was written.
//
// A fresh read, because lowering writes the chain into a document rather than
// building one: routing, snapshots and controller assignments are whatever
// the file already held.
func (s *RoundTripPublicTestSuite) roundTrip(path string) []byte {
	raw := s.read(path)

	doc, err := preset.Read(bytes.NewReader(raw))
	s.Require().NoError(err)

	spec, err := lift.Lift(doc, s.cat)
	s.Require().NoError(err)

	back, err := preset.Read(bytes.NewReader(raw))
	s.Require().NoError(err)
	s.Require().NoError(lift.Lower(back, spec, s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, back))

	return out.Bytes()
}

// canonical re-encodes JSON with sorted keys and no whitespace, so two
// documents saying the same thing compare equal whatever their layout.
func (s *RoundTripPublicTestSuite) canonical(raw []byte) string {
	var v any
	s.Require().NoError(json.Unmarshal(raw, &v))

	out, err := json.Marshal(v)
	s.Require().NoError(err)

	return string(out)
}

func (s *RoundTripPublicTestSuite) read(path string) []byte {
	raw, err := os.ReadFile(path) //nolint:gosec // a path this suite walked
	s.Require().NoError(err)

	return raw
}

// fixtures are the presets committed with this test.
func (s *RoundTripPublicTestSuite) fixtures() []string {
	found, err := filepath.Glob(filepath.Join("testdata", "*.hlx"))
	s.Require().NoError(err)
	s.Require().NotEmpty(found)

	return found
}

// corpus lists every preset in the uncommitted body of real ones.
func (s *RoundTripPublicTestSuite) corpus() []string {
	var found []string

	err := filepath.Walk(corpusDir, func(p string, i os.FileInfo, e error) error {
		if e == nil && !i.IsDir() && strings.EqualFold(filepath.Ext(p), ".hlx") {
			found = append(found, p)
		}

		return nil
	})
	s.Require().NoError(err)

	return found
}

func TestRoundTripPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RoundTripPublicTestSuite))
}
