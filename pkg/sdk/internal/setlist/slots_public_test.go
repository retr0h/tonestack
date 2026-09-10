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
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/setlist"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

type SlotsPublicTestSuite struct {
	suite.Suite
}

func (s *SlotsPublicTestSuite) doc(name string) *setlist.Document {
	f, err := os.Open(filepath.Join("testdata", name)) //nolint:gosec // a test fixture
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := setlist.Read(f)
	s.Require().NoError(err)

	return doc
}

// TestCopy writes one slot over another.
func (s *SlotsPublicTestSuite) TestCopy() {
	tests := []struct {
		name string
		from int
		to   int
		// what the first two slots must hold afterwards.
		want []string
		err  bool
	}{
		{
			name: "a slot over another",
			to:   1,
			want: []string{"First", "First"},
		},
		{name: "a source that is not there", from: 99, to: 1, err: true},
		{name: "a destination that is not there", to: 99, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.doc("setlist.hls")

			err := doc.Copy(
				setlist.Address{Slot: tt.from}, setlist.Address{Slot: tt.to})

			if tt.err {
				s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)

				return
			}

			s.Require().NoError(err)

			for i, want := range tt.want {
				s.Require().Equal(want, doc.Setlists[0].Slots[i].Meta.Name)
			}

			if len(tt.want) < 2 {
				return
			}

			// A struct copy would leave the two slots sharing their tone, so
			// the next edit to the destination would rewrite the source too.
			before := len(doc.Setlists[0].Slots[tt.from].Tone)
			doc.Setlists[0].Slots[tt.to].Tone["dsp9"] = preset.Tone{}

			s.Require().Len(doc.Setlists[0].Slots[tt.from].Tone, before,
				"the source shares nothing with the copy")
		})
	}
}

// TestSwap exchanges two slots.
func (s *SlotsPublicTestSuite) TestSwap() {
	tests := []struct {
		name string
		from int
		to   int
		// swap twice, which must leave the setlist as it was.
		twice bool
		want  []string
		err   bool
	}{
		{
			name: "two slots",
			to:   1,
			want: []string{"Second", "First"},
		},
		{
			name:  "the same two twice",
			to:    3,
			twice: true,
			want:  []string{"First", "Second"},
		},
		{name: "a first address that is not there", from: 99, to: 1, err: true},
		{name: "a second that is not there", to: 99, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.doc("setlist.hls")
			a, b := setlist.Address{Slot: tt.from}, setlist.Address{Slot: tt.to}

			err := doc.Swap(a, b)

			if tt.err {
				s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)

				return
			}

			s.Require().NoError(err)

			if tt.twice {
				s.Require().NoError(doc.Swap(a, b))
			}

			for i, want := range tt.want {
				s.Require().Equal(want, doc.Setlists[0].Slots[i].Meta.Name)
			}
		})
	}
}

// TestRename names a slot.
func (s *SlotsPublicTestSuite) TestRename() {
	tests := []struct {
		name string
		slot int
		err  bool
	}{
		{name: "a slot the setlist has", slot: 2},
		{name: "one it does not", slot: 99, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.doc("setlist.hls")

			err := doc.Rename(setlist.Address{Slot: tt.slot}, "Renamed")

			if tt.err {
				s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal("Renamed", doc.Setlists[0].Slots[tt.slot].Meta.Name)
		})
	}
}

// TestNames lists what a setlist holds.
func (s *SlotsPublicTestSuite) TestNames() {
	tests := []struct {
		name    string
		setlist int
		want    []string
		err     bool
	}{
		{
			name: "a setlist the file has",
			want: []string{"First", "Second", "New Preset", "Unknown Gear"},
		},
		{name: "one it does not", setlist: 9, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.doc("setlist.hls").Names(tt.setlist)

			if tt.err {
				s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestName reads what a setlist calls itself.
func (s *SlotsPublicTestSuite) TestName() {
	tests := []struct {
		name string
		meta string
		want string
	}{
		{name: "metadata naming it", meta: `{"name":"Songs"}`, want: "Songs"},
		{name: "metadata with no name", meta: `{}`},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			sl := setlist.Setlist{Meta: json.RawMessage(tt.meta)}

			s.Require().Equal(tt.want, sl.Name())
		})
	}
}

// TestWrite writes a setlist back out.
func (s *SlotsPublicTestSuite) TestWrite() {
	tests := []struct {
		name string
		file string
		// claim the file holds one setlist when it holds several.
		mislabelled bool
		// a payload nothing can encode.
		unencodable bool
		deaf        bool
		errText     string
	}{
		{name: "a setlist as it was read", file: "setlist.hls"},
		{
			name:        "a bundle claiming to be one setlist",
			file:        "bundle.hlb",
			mislabelled: true,
			errText:     "holds one setlist, not 2",
		},
		{
			name:        "a payload that cannot encode",
			file:        "setlist.hls",
			unencodable: true,
			errText:     "encoding payload",
		},
		{
			name:    "a writer that fails",
			file:    "setlist.hls",
			deaf:    true,
			errText: "encoding setlist",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.doc(tt.file)

			if tt.mislabelled {
				doc.Schema = setlist.SchemaSetlist
			}

			if tt.unencodable {
				// A raw message that is not JSON has no representation, so
				// this fails where a real payload never would.
				doc.Setlists[0].Slots[0].Tone = map[string]preset.Tone{
					"dsp0": {"block0": json.RawMessage("not json")},
				}
			}

			var out bytes.Buffer

			w := io.Writer(&out)
			if tt.deaf {
				w = &failingWriter{}
			}

			err := setlist.Write(w, doc)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(out.String())
		})
	}
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestSlotsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotsPublicTestSuite))
}
