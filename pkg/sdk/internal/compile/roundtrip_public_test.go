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

package compile_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// corpusDir holds thousands of presets other people made. It is not committed
// — see docs/knowledge.md — so the sweep over it runs only where it exists,
// and the committed fixtures carry the same assertions everywhere else.
const corpusDir = "../../resources/schemas/corpus"

type RoundTripPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *RoundTripPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// TestLiftAndLower covers the three claims the format rests on, over every
// preset committed with this test.
//
// A preset read into a rig and written back must say the same thing. A rig
// built into a preset and read back must be the rig that went in — anything
// RigSpec models but does not write is invisible to the first claim and
// obvious in the second. And a rig must rebuild its preset with the original
// gone, which is the path a shared rig takes: it reaches somebody else
// without the preset it came from, so lowering into that preset proves
// nothing about what the rig carries.
func (s *RoundTripPublicTestSuite) TestLiftAndLower() {
	for _, path := range s.fixtures() {
		s.Run(filepath.Base(path), func() {
			raw := s.canonical(s.read(path))

			s.Require().Equal(raw, s.canonical(s.roundTrip(path)),
				"a preset read into a rig and written back must say the same thing")

			s.Require().Equal(raw, s.canonical(s.fromNothing(path)),
				"a rig must rebuild its preset with the original gone")

			was, now := s.backAgain(s.read(path))
			s.Require().Equal(was, now,
				"every field a rig models must be written as well as read")
		})
	}
}

// TestLift records what a block actually was.
func (s *RoundTripPublicTestSuite) TestLift() {
	// A gear name does not identify a model: 665 of them share 469 names, and
	// "Ampeg SVT" matches both channels. Without the identifier a rig rebuilds
	// into a different preset.
	doc, err := preset.Read(bytes.NewReader(s.read(s.fixtures()[0])))
	s.Require().NoError(err)

	spec, err := compile.Lift(doc, s.cat)
	s.Require().NoError(err)
	s.Require().NotEmpty(spec.Chain)

	for _, entry := range spec.Chain {
		s.Require().NotNil(entry.Models, "every block records what it was")
		s.Require().Contains(*entry.Models, s.cat.Device)
		s.Require().NotEmpty(entry.Gear, "and what a person would call it")
	}
}

func (s *RoundTripPublicTestSuite) TestTheWholeCorpusSurvivesIt() {
	// The directory itself is committed — the gitignore keeps a little
	// metadata in it — so its presence says nothing. What matters is whether
	// anybody has fetched the presets.
	found := s.corpus()
	if len(found) == 0 {
		s.T().Skip("no corpus fetched; the committed fixtures cover the same ground")
	}

	var (
		same, differ, skipped int
		failed                []string
	)

	for _, path := range found {
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

		was, now := s.backAgain(raw)
		if s.canonical(raw) == s.canonical(s.roundTrip(path)) &&
			s.canonical(raw) == s.canonical(s.fromNothing(path)) &&
			was == now {
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
		"every preset must survive becoming a rig and being written back — "+
			"into itself, into nothing, and rig to preset to rig; "+
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

	back, err := preset.Read(bytes.NewReader(raw))
	s.Require().NoError(err)

	return s.through(raw, back)
}

// fromNothing is the path a shared rig takes: read a preset, keep only the
// rig, and build a preset out of it with the original gone.
//
// This is the claim that matters. Lowering into the preset a rig came from
// proves little — routing and snapshots survive because nobody removed them.
// A rig that reaches somebody else arrives on its own.
func (s *RoundTripPublicTestSuite) fromNothing(path string) []byte {
	blank, err := preset.Blank()
	s.Require().NoError(err)

	return s.through(s.read(path), blank)
}

// backAgain lifts a preset, builds it, and lifts what was built.
//
// The other direction, and the one that catches a field which reads but never
// writes: such a field survives .hlx to .hlx because the preset underneath
// still holds it, and disappears here because the rig is all there is. Both
// rigs must say the same thing.
func (s *RoundTripPublicTestSuite) backAgain(raw []byte) (string, string) {
	from, err := preset.Read(bytes.NewReader(raw))
	s.Require().NoError(err)

	first, err := compile.Lift(from, s.cat)
	s.Require().NoError(err)

	blank, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(compile.Lower(blank, first, s.cat))

	second, err := compile.Lift(blank, s.cat)
	s.Require().NoError(err)

	return s.marshal(first), s.marshal(second)
}

// marshal renders a rig for comparison, as JSON rather than as the YAML it is
// written in.
//
// Go sorts a map's keys when it encodes JSON; the YAML writer orders them
// differently for the same content, which would make this test fail over how
// a document was laid out rather than over what it says.
func (s *RoundTripPublicTestSuite) marshal(spec riggen.RigSpec) string {
	body, err := json.Marshal(spec)
	s.Require().NoError(err)

	return s.canonical(body)
}

// through lifts raw to a rig and lowers that rig into doc.
func (s *RoundTripPublicTestSuite) through(raw []byte, doc *preset.Document) []byte {
	from, err := preset.Read(bytes.NewReader(raw))
	s.Require().NoError(err)

	spec, err := compile.Lift(from, s.cat)
	s.Require().NoError(err)

	s.Require().NoError(compile.Lower(doc, spec, s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))

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

// corpus lists every preset in the body of real ones, which is fetched rather
// than committed.
func (s *RoundTripPublicTestSuite) corpus() []string {
	var found []string

	// Errors are ignored rather than asserted: an absent corpus is the
	// ordinary case away from a machine that has fetched one.
	_ = filepath.Walk(corpusDir, func(p string, i os.FileInfo, e error) error {
		if e == nil && !i.IsDir() && strings.EqualFold(filepath.Ext(p), ".hlx") {
			found = append(found, p)
		}

		return nil
	})

	return found
}

func TestRoundTripPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RoundTripPublicTestSuite))
}
