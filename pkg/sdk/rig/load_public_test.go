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

package rig_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/gen"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

type LoadPublicTestSuite struct {
	suite.Suite
}

const smallest = `
schema: RigSpec
id: mike-dirnt
subject: { kind: artist, name: Mike Dirnt, band: Green Day }
instrument: bass
chain:
  - { role: amp, gear: Ampeg SVT }
`

// TestLoad reads a rig off a reader.
func (s *LoadPublicTestSuite) TestLoad() {
	tests := []struct {
		name string
		in   string
		// a reader that fails outright.
		deaf    bool
		errText string
	}{
		{name: "the smallest rig there is", in: smallest},
		{
			name:    "something that is not YAML",
			in:      "\tnope: [",
			errText: "decoding rig",
		},
		{
			name:    "a preset, which is a different kind of document",
			in:      `{"schema":"L6Preset","version":6}`,
			errText: "not a valid rig",
		},
		{
			// Decoding drops what the types have no field for, so this used
			// to pass with the misspelt line quietly gone.
			name: "a field nobody spelled right",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"tecnique: pick\n",
			errText: `property "tecnique" is unsupported`,
		},
		{
			name: "a field nobody spelled right, inside the chain",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n" +
				"  - {role: amp, gear: Ampeg SVT, gera: nonsense}\n",
			errText: `property "gera" is unsupported`,
		},
		{
			// technique was a sentence until the three things it says were
			// separated. Two files in this repository carried the old
			// spelling and nothing outside it did, so it is refused rather
			// than accepted alongside the new one.
			name: "technique as the sentence it used to be",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"technique: pick, near the bridge\n",
			errText: "technique",
		},
		{
			name: "a way of playing the contract does not name",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"technique: {attack: plectrum}\n",
			errText: `technique.attack value is not one of the allowed values`,
		},
		{
			name: "a technique saying nothing about the attack",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"technique: {position: bridge}\n",
			errText: `property "attack" is missing`,
		},
		{
			// character was a list of bare strings until each term had to
			// carry why it is believed. Nothing outside this repository
			// writes a rig, so the old spelling is refused rather than read
			// alongside the new one.
			name: "character as the bare list it used to be",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"character: [mid-forward]\n",
			errText: "character",
		},
		{
			name: "a character term saying nothing",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"character:\n  - evidence: [{kind: llm}]\n",
			errText: `property "term" is missing`,
		},
		{
			// A claim about how somebody plays is asserted or it is watched,
			// and the format has to be able to say which.
			name: "technique carrying why it is believed",
			in: smallest + "technique:\n  attack: pick\n  evidence:\n" +
				"    - {kind: video, url: \"https://x.test/v\", at: \"1:42\"}\n",
		},
		{
			name: "a character term carrying why it is believed",
			in: smallest +
				"character:\n  - term: mid-forward\n    evidence: [{kind: llm}]\n",
		},
		{
			// The contract calls this an integer and Go's int cannot hold
			// it, so the decode fails on a document that validated. It came
			// back as a rig with the field zeroed and no error at all.
			name: "a number larger than the type that holds it",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"controllers:\n" +
				"  - {controller: 99999999999999999999, block: 0, parameter: Drive}\n",
			errText: "of type int",
		},
		{
			// One version, so a rig stating another is refused rather than
			// read as if its fields meant the same thing.
			name: "a version this contract is not",
			in: "schema: RigSpec\nversion: 3\nid: x\n" +
				"subject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n",
			errText: "version",
		},
		{
			name: "a link that is not one",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"evidence:\n  - {kind: cited, url: mikes-website}\n",
			errText: "evidence[0].url",
		},
		{
			// Where in a recording, so it has to be a time.
			name: "a place in a recording, given in words",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"evidence:\n  - {kind: video, url: \"https://x.test/v\", at: the end}\n",
			errText: "evidence[0].at",
		},
		{
			name: "a correction dated in words",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"mutations:\n  - {at: yesterday, ask: make it clunkier}\n",
			errText: "mutations[0].at",
		},
		{
			// A path into this document, which is how a correction says what it
			// moved.
			name: "a path nothing could follow",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain:\n  - {role: amp, gear: Ampeg SVT}\n" +
				"mutations:\n  - {at: \"2026-09-06\", ask: x,\n" +
				"     changed: [{path: the drive knob, from: 1, to: 2}]}\n",
			errText: "changed[0].path",
		},
		{
			name: "a rig holding no chain",
			in: "schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain: []\n",
			errText: "chain minimum number of items is 1",
		},
		{name: "a reader that fails", deaf: true, errText: "reading rig"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			in := io.Reader(strings.NewReader(tt.in))
			if tt.deaf {
				in = &failingReader{}
			}

			got, err := rig.Load(in)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal("mike-dirnt", got.ID)
			s.Require().Equal("Mike Dirnt", got.Subject.Name)
			s.Require().Equal(gen.InstrumentBass, got.Instrument)
			s.Require().Len(got.Chain, 1)
			s.Require().Equal("Ampeg SVT", got.Chain[0].Gear)
		})
	}
}

// TestWrite writes a rig back out.
func (s *LoadPublicTestSuite) TestWrite() {
	tests := []struct {
		name string
		// a rig read from this document, or the zero rig, which is not one.
		in      string
		deaf    bool
		err     error
		errText string
	}{
		{name: "a rig, which reads back as itself", in: smallest},
		{name: "a rig that is not one", err: rig.ErrInvalid},
		{
			name:    "a writer that fails",
			in:      smallest,
			deaf:    true,
			errText: "writing rig",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var spec gen.RigSpec

			if tt.in != "" {
				got, err := rig.Load(strings.NewReader(tt.in))
				s.Require().NoError(err)

				spec = got
			}

			var buf bytes.Buffer

			out := io.Writer(&buf)
			if tt.deaf {
				out = &failingWriter{}
			}

			err := rig.Write(out, spec)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			again, err := rig.Load(&buf)
			s.Require().NoError(err)
			s.Require().Equal(spec, again)
		})
	}
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// TestTheContractAcceptsTheVersionThisPackageWrites keeps the constant and
// the contract from drifting apart.
//
// rig.Version says which version this package reads and writes; the contract
// says which one a document may state. Nothing else compares them, and a bump
// that moved one and not the other would be silent.
func (s *LoadPublicTestSuite) TestTheContractAcceptsTheVersionThisPackageWrites() {
	stated := fmt.Sprintf("version: %d\n", rig.Version)

	_, err := rig.Load(strings.NewReader(stated + smallest))

	s.Require().NoError(err)
}

func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
