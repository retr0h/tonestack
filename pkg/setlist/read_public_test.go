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

package setlist_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/setlist"
)

type ReadPublicTestSuite struct {
	suite.Suite
}

func (s *ReadPublicTestSuite) read(name string) (*setlist.Document, error) {
	f, err := os.Open(filepath.Join("testdata", name)) //nolint:gosec // a test fixture
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	return setlist.Read(f)
}

func (s *ReadPublicTestSuite) TestReadDecodesASetlist() {
	doc, err := s.read("setlist.hls")

	s.Require().NoError(err)
	s.Require().Equal(setlist.SchemaSetlist, doc.Schema)
	s.Require().Equal(2, doc.Version)
	s.Require().Len(doc.Setlists, 1)
	s.Require().Equal("Test Setlist", doc.Setlists[0].Name())
	s.Require().Len(doc.Setlists[0].Slots, 4)
	s.Require().Equal("First", doc.Setlists[0].Slots[0].Meta.Name)
}

func (s *ReadPublicTestSuite) TestReadDecodesABundle() {
	doc, err := s.read("bundle.hlb")

	s.Require().NoError(err)
	s.Require().Equal(setlist.SchemaBundle, doc.Schema)
	s.Require().Len(doc.Setlists, 2)
	s.Require().Equal("Second Setlist", doc.Setlists[1].Name())
	s.Require().Equal("Only", doc.Setlists[1].Slots[0].Meta.Name)
}

func (s *ReadPublicTestSuite) TestReadRejects() {
	tests := []struct {
		name    string
		fixture string
		want    error
		message string
	}{
		{"a preset", "notasetlist.hls", setlist.ErrNotASetlist, "schema is"},
		{"a file with no schema", "noschema.hls", setlist.ErrNotASetlist, "no schema field"},
		{"something that is not JSON", "notjson.hls", nil, "decoding setlist"},
		{"a bad checksum", "badcrc.hls", setlist.ErrCorrupt, "checksum"},
		{"a bad size", "badsize.hls", setlist.ErrCorrupt, "decompressed size"},
		{"data that is not base64", "badbase64.hls", nil, "decoding encoded_data"},
		{"data that is not zlib", "badzlib.hls", nil, "opening compressed payload"},
		{"a truncated stream", "truncated.hls", nil, "decompressing payload"},
		{"a payload that is not JSON", "badpayload.hls", nil, "decoding setlist payload"},
		{
			"a bundle payload that is not JSON", "badbundlepayload.hlb",
			nil, "decoding bundle payload",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := s.read(tc.fixture)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)

			if tc.want != nil {
				s.Require().ErrorIs(err, tc.want)
			}
		})
	}
}

func (s *ReadPublicTestSuite) TestSlot() {
	doc, err := s.read("setlist.hls")
	s.Require().NoError(err)

	got, err := doc.Slot(0, 1)
	s.Require().NoError(err)
	s.Require().Equal("Second", got.Meta.Name)

	tests := []struct {
		name     string
		sl, slot int
	}{
		{"a setlist that is not there", 9, 0},
		{"a negative setlist", -1, 0},
		{"a slot that is not there", 0, 99},
		{"a negative slot", 0, -1},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := doc.Slot(tc.sl, tc.slot)

			s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)
			s.Require().Contains(err.Error(), "no such slot")
		})
	}
}

func (s *ReadPublicTestSuite) TestRoundTrip() {
	for _, name := range []string{"setlist.hls", "bundle.hlb"} {
		s.Run(name, func() {
			doc, err := s.read(name)
			s.Require().NoError(err)

			var buf bytes.Buffer
			s.Require().NoError(setlist.Write(&buf, doc))

			again, err := setlist.Read(bytes.NewReader(buf.Bytes()))
			s.Require().NoError(err)
			s.Require().Equal(doc, again)
		})
	}
}

func TestReadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadPublicTestSuite))
}
