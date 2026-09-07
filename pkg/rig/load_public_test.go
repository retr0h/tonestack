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

func (s *LoadPublicTestSuite) TestReadsARig() {
	got, err := rig.Load(strings.NewReader(smallest))

	s.Require().NoError(err)
	s.Require().Equal("mike-dirnt", got.ID)
	s.Require().Equal("Mike Dirnt", got.Subject.Name)
	s.Require().Equal(gen.InstrumentBass, got.Instrument)
	s.Require().Len(got.Chain, 1)
	s.Require().Equal("Ampeg SVT", got.Chain[0].Gear)
}

func (s *LoadPublicTestSuite) TestWhatIsWrittenReadsBack() {
	first, err := rig.Load(strings.NewReader(smallest))
	s.Require().NoError(err)

	var out bytes.Buffer
	s.Require().NoError(rig.Write(&out, first))

	again, err := rig.Load(&out)

	s.Require().NoError(err)
	s.Require().Equal(first, again)
}

func (s *LoadPublicTestSuite) TestRefusesWhatIsNotARig() {
	tests := []struct {
		name    string
		in      string
		message string
	}{
		{"something that is not YAML", "\tnope: [", "decoding rig"},
		{
			"a preset, which is a different kind of document",
			`{"schema":"L6Preset","version":6}`,
			"not a valid rig",
		},
		{
			"a rig holding no chain",
			"schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n" +
				"instrument: bass\nchain: []\n",
			"chain holds nothing",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := rig.Load(strings.NewReader(tc.in))

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *LoadPublicTestSuite) TestRefusesToWriteOneThatIsNotValid() {
	err := rig.Write(&bytes.Buffer{}, gen.RigSpec{})

	s.Require().ErrorIs(err, rig.ErrInvalid)
}

func (s *LoadPublicTestSuite) TestReportsAReaderThatFails() {
	_, err := rig.Load(&failingReader{})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reading rig")
}

func (s *LoadPublicTestSuite) TestReportsAWriterThatFails() {
	spec, err := rig.Load(strings.NewReader(smallest))
	s.Require().NoError(err)

	err = rig.Write(&failingWriter{}, spec)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing rig")
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
