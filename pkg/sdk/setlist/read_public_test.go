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

	"github.com/retr0h/tonestack/pkg/sdk/setlist"
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

// TestRead decodes a setlist or a bundle.
func (s *ReadPublicTestSuite) TestRead() {
	tests := []struct {
		name    string
		fixture string
		// what the document must say once it is read.
		schema    string
		version   int
		setlists  int
		named     map[int]string
		slots     map[int]int
		firstSlot map[int]string

		err     error
		errText string
	}{
		{
			name:      "a setlist",
			fixture:   "setlist.hls",
			schema:    setlist.SchemaSetlist,
			version:   2,
			setlists:  1,
			named:     map[int]string{0: "Test Setlist"},
			slots:     map[int]int{0: 4},
			firstSlot: map[int]string{0: "First"},
		},
		{
			name:      "a bundle holding several",
			fixture:   "bundle.hlb",
			schema:    setlist.SchemaBundle,
			version:   1,
			setlists:  2,
			named:     map[int]string{1: "Second Setlist"},
			firstSlot: map[int]string{1: "Only"},
		},
		{
			name:    "a preset",
			fixture: "notasetlist.hls",
			err:     setlist.ErrNotASetlist,
			errText: "schema is",
		},
		{
			name:    "a file with no schema",
			fixture: "noschema.hls",
			err:     setlist.ErrNotASetlist,
			errText: "no schema field",
		},
		{
			name:    "something that is not JSON",
			fixture: "notjson.hls",
			errText: "decoding setlist",
		},
		{
			name:    "a bad checksum",
			fixture: "badcrc.hls",
			err:     setlist.ErrCorrupt,
			errText: "checksum",
		},
		{
			name:    "a bad size",
			fixture: "badsize.hls",
			err:     setlist.ErrCorrupt,
			errText: "decompressed size",
		},
		{
			name:    "data that is not base64",
			fixture: "badbase64.hls",
			errText: "decoding encoded_data",
		},
		{
			name:    "data that is not zlib",
			fixture: "badzlib.hls",
			errText: "opening compressed payload",
		},
		{
			name:    "a truncated stream",
			fixture: "truncated.hls",
			errText: "decompressing payload",
		},
		{
			name:    "a payload that is not JSON",
			fixture: "badpayload.hls",
			errText: "decoding setlist payload",
		},
		{
			name:    "a bundle payload that is not JSON",
			fixture: "badbundlepayload.hlb",
			errText: "decoding bundle payload",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.read(tt.fixture)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.schema, got.Schema)
			s.Require().Equal(tt.version, got.Version)
			s.Require().Len(got.Setlists, tt.setlists)

			for at, want := range tt.named {
				s.Require().Equal(want, got.Setlists[at].Name())
			}

			for at, want := range tt.slots {
				s.Require().Len(got.Setlists[at].Slots, want)
			}

			for at, want := range tt.firstSlot {
				s.Require().Equal(want, got.Setlists[at].Slots[0].Meta.Name)
			}
		})
	}
}

// TestSlot addresses one slot of one setlist.
func (s *ReadPublicTestSuite) TestSlot() {
	tests := []struct {
		name     string
		sl, slot int
		want     string
	}{
		{name: "a slot the setlist holds", slot: 1, want: "Second"},
		{name: "a setlist that is not there", sl: 9},
		{name: "a negative setlist", sl: -1},
		{name: "a slot that is not there", slot: 99},
		{name: "a negative slot", slot: -1},
	}

	doc, err := s.read("setlist.hls")
	s.Require().NoError(err)

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := doc.Slot(tt.sl, tt.slot)

			if tt.want == "" {
				s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)
				s.Require().Contains(err.Error(), "no such slot")

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Meta.Name)
		})
	}
}

// TestRoundTrip is the claim a file rests on: what was read is what is
// written back.
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
