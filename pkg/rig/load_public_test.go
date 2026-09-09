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
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/rig"
	"github.com/retr0h/tonestack/pkg/rig/gen"
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

func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
