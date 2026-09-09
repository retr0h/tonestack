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

package wire_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// DocumentPublicTestSuite covers reading a preset and writing it back.
//
// This is the part that corrupts hardware when it is wrong. A device seeks by
// a table of byte offsets rather than walking the document, so a re-encode
// that shifts any length points every later offset at the wrong thing. The
// device accepts such a write and then reads the preset as empty, and two
// other projects lost sessions to exactly that.
type DocumentPublicTestSuite struct {
	suite.Suite
}

// capture returns one slot as an HX Stomp actually sent it.
func (s *DocumentPublicTestSuite) capture(name string) []byte {
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	s.Require().NoError(err)

	return raw
}

// offsets reads the table a preset carries.
func (s *DocumentPublicTestSuite) offsets(raw []byte) []uint32 {
	dec := msgpack.NewDecoder(bytes.NewReader(raw))

	_, err := dec.DecodeRaw()
	s.Require().NoError(err)

	table, err := dec.DecodeRaw()
	s.Require().NoError(err)

	// Past the str16 length prefix the device writes it with.
	table = table[3:]

	out := make([]uint32, 0, len(table)/4)
	for i := 0; i+4 <= len(table); i += 4 {
		out = append(out, binary.LittleEndian.Uint32(table[i:]))
	}

	return out
}

// header returns a preset header with an empty offset table.
func header(magic string) []byte {
	return append([]byte(magic+"\xda\x00\x30"), make([]byte, 48)...)
}

// TestDecodeDocument reads what a device sent.
func (s *DocumentPublicTestSuite) TestDecodeDocument() {
	tests := []struct {
		name string
		file string
		body []byte
		err  bool
	}{
		// Byte for byte. Anything less than this must never reach a device.
		{name: "a preset", file: "preset.bin"},
		{name: "one with switch colours", file: "switches.bin"},
		{name: "a slot holding nothing", file: "empty.bin"},
		{
			// The length of a string can be written three ways and a device
			// uses one of them. Reading is not the place to be strict about
			// which.
			name: "a magic written another way",
			body: append(header("\xd9\x09l6-helix\x00"), 0x80),
		},
		{name: "nothing at all", err: true},
		{name: "something else entirely", body: []byte{0xc0}, err: true},
		{
			name: "the header and no more",
			body: []byte("\xa9l6-helix\x00"),
			err:  true,
		},
		{
			name: "an offset table of the wrong size",
			body: []byte("\xa9l6-helix\x00\xa1x"),
			err:  true,
		},
		{
			name: "a header with no document",
			body: header("\xa9l6-helix\x00"),
			err:  true,
		},
		{
			name: "a section with no key",
			body: append(header("\xa9l6-helix\x00"), 0x81, 0xc0),
			err:  true,
		},
		{
			name: "a key with no section",
			body: append(header("\xa9l6-helix\x00"), 0x81, 0x00),
			err:  true,
		},
		{
			name: "a key that is not a number",
			body: append(header("\xa9l6-helix\x00"), 0x81, 0xa1, 'x', 0xc0),
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			raw := tt.body
			if tt.file != "" {
				raw = s.capture(tt.file)
			}

			doc, err := wire.DecodeDocument(raw)

			if tt.err {
				s.Require().ErrorIs(err, wire.ErrNotADocument)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(doc)

			if tt.file != "" {
				s.Require().Equal(raw, doc.Encode())
			}
		})
	}
}

// TestEncode writes a preset back, table and all.
//
// The table is an index into the document: the first entry is the map, the
// last two are the end, and each one between points at the key byte of a
// section. It is rewritten rather than carried, because a section that grows
// moves everything after it, and an offset left pointing at where something
// used to be is how a preset reads back empty.
func (s *DocumentPublicTestSuite) TestEncode() {
	tests := []struct {
		name string
		grow bool
	}{
		{name: "a preset as the device wrote it"},
		{name: "a preset with a section that grew", grow: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			raw := s.capture("preset.bin")

			doc, err := wire.DecodeDocument(raw)
			s.Require().NoError(err)

			if tt.grow {
				body, ok := doc.Section(7)
				s.Require().True(ok, "a preset records what wrote it")

				doc.SetSection(7, append(append(msgpack.RawMessage(nil), body...),
					msgpack.RawMessage("\xc0")...))
			}

			got := doc.Encode()
			s.Require().Equal(tt.grow, len(got) > len(raw))

			offsets := s.offsets(got)
			s.Require().Len(offsets, len(s.offsets(raw)))

			// The document is however long it is, and the table says so.
			s.Require().Equal(uint32(len(got)), offsets[len(offsets)-1])

			s.Require().Equal(byte(0x89), got[offsets[0]],
				"the preset map, nine sections")

			for i, key := range []byte{0, 1, 3, 4, 2, 5, 6, 7, 10} {
				s.Require().Equal(key, got[offsets[i+1]],
					"offset %d names section %d", i+1, key)
			}
		})
	}
}

// TestSetSection puts a section into a preset that did not have one.
func (s *DocumentPublicTestSuite) TestSetSection() {
	tests := []struct {
		name  string
		first int8
		last  int8
	}{
		{name: "a section the preset lacked", first: 9, last: 9},
		{
			// A fixmap runs to fifteen entries. Past that the length is
			// written differently, and a document that got it wrong would not
			// decode at all.
			name:  "more sections than a fixmap holds",
			first: 20,
			last:  31,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := wire.DecodeDocument(s.capture("preset.bin"))
			s.Require().NoError(err)

			_, had := doc.Section(tt.last)
			s.Require().False(had)

			for key := tt.first; key <= tt.last; key++ {
				doc.SetSection(key, msgpack.RawMessage("\xc3"))
			}

			again, err := wire.DecodeDocument(doc.Encode())
			s.Require().NoError(err)

			body, ok := again.Section(tt.last)
			s.Require().True(ok)
			s.Require().Equal(msgpack.RawMessage("\xc3"), body)
		})
	}
}

// TestNewDocument rebuilds a preset holding fewer sections.
//
// The table has an entry for every section a preset can hold, and a preset
// need not hold them all. An offset for a section that is not there is left
// alone rather than pointed somewhere wrong.
func (s *DocumentPublicTestSuite) TestNewDocument() {
	doc, err := wire.DecodeDocument(s.capture("preset.bin"))
	s.Require().NoError(err)

	// One section dropped, which is what an older firmware writing fewer of
	// them would look like.
	again, err := wire.DecodeDocument(wire.NewDocument(doc, []int8{0, 7}).Encode())
	s.Require().NoError(err)

	_, ok := again.Section(0)
	s.Require().True(ok)

	_, ok = again.Section(3)
	s.Require().False(ok, "the section that was dropped")
}

func TestDocumentPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DocumentPublicTestSuite))
}
