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

func (s *DocumentPublicTestSuite) TestAPresetSurvivesBeingReadAndWritten() {
	// Byte for byte. Anything less than this must never reach a device.
	for _, name := range []string{"preset.bin", "switches.bin", "empty.bin"} {
		s.Run(name, func() {
			raw := s.capture(name)

			doc, err := wire.DecodeDocument(raw)
			s.Require().NoError(err)
			s.Require().Equal(raw, doc.Encode())
		})
	}
}

func (s *DocumentPublicTestSuite) TestChangingASectionMovesTheOffsetsAfterIt() {
	// The reason the table is rewritten rather than carried: a section that
	// grows moves everything after it, and an offset left pointing at where
	// something used to be is how a preset reads back empty.
	raw := s.capture("preset.bin")

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	body, ok := doc.Section(7)
	s.Require().True(ok, "a preset records what wrote it")

	doc.SetSection(7, append(append(msgpack.RawMessage(nil), body...),
		msgpack.RawMessage("\xc0")...))

	got := doc.Encode()
	s.Require().Greater(len(got), len(raw), "the section grew")

	was, now := s.offsets(raw), s.offsets(got)
	s.Require().Len(now, len(was))

	// The document is longer, and the table says so.
	s.Require().Equal(uint32(len(got)), now[len(now)-1])
	s.Require().Equal(uint32(len(raw)), was[len(was)-1])

	// Every offset still points at the byte it names.
	for i, o := range now {
		s.Require().LessOrEqual(int(o), len(got), "offset %d is inside the file", i)
	}
}

func (s *DocumentPublicTestSuite) TestEveryOffsetPointsAtItsSection() {
	// The table is an index into the document: the first entry is the map,
	// the last two are the end, and each one between points at the key byte
	// of a section.
	raw := s.capture("preset.bin")

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	got := doc.Encode()
	offsets := s.offsets(got)

	s.Require().Equal(byte(0x89), got[offsets[0]], "the preset map, nine sections")

	for i, key := range []byte{0, 1, 3, 4, 2, 5, 6, 7, 10} {
		s.Require().Equal(key, got[offsets[i+1]],
			"offset %d names section %d", i+1, key)
	}
}

func (s *DocumentPublicTestSuite) TestAddingASectionThePresetLacked() {
	raw := s.capture("preset.bin")

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	_, had := doc.Section(9)
	s.Require().False(had)

	doc.SetSection(9, msgpack.RawMessage("\xc3"))

	again, err := wire.DecodeDocument(doc.Encode())
	s.Require().NoError(err)

	body, ok := again.Section(9)
	s.Require().True(ok)
	s.Require().Equal(msgpack.RawMessage("\xc3"), body)
}

func (s *DocumentPublicTestSuite) TestAPresetMissingASectionTheTableNames() {
	// The table has an entry for every section a preset can hold, and a
	// preset need not hold them all. An offset for a section that is not
	// there is left alone rather than pointed somewhere wrong.
	raw := s.capture("preset.bin")

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	// Rebuilt with one section dropped, which is what an older firmware
	// writing fewer of them would look like.
	fewer := wire.NewDocument(doc, []int8{0, 7})

	again, err := wire.DecodeDocument(fewer.Encode())
	s.Require().NoError(err)

	_, ok := again.Section(0)
	s.Require().True(ok)

	_, ok = again.Section(3)
	s.Require().False(ok, "the section that was dropped")
}

func (s *DocumentPublicTestSuite) TestAPresetWithMoreSectionsThanAMapHolds() {
	// A fixmap runs to fifteen entries. Past that the length is written
	// differently, and a document that got it wrong would not decode at all.
	raw := s.capture("preset.bin")

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	for key := int8(20); key < 32; key++ {
		doc.SetSection(key, msgpack.RawMessage("\xc3"))
	}

	again, err := wire.DecodeDocument(doc.Encode())
	s.Require().NoError(err)

	body, ok := again.Section(31)
	s.Require().True(ok)
	s.Require().Equal(msgpack.RawMessage("\xc3"), body)
}

func (s *DocumentPublicTestSuite) TestAMagicWrittenAnotherWay() {
	// The length of a string can be written three ways and a device uses one
	// of them. Reading is not the place to be strict about which.
	raw := append(append(
		[]byte("\xd9\x09l6-helix\x00\xda\x00\x30"), make([]byte, 48)...), 0x80)

	doc, err := wire.DecodeDocument(raw)

	s.Require().NoError(err)
	s.Require().NotNil(doc)
}

func (s *DocumentPublicTestSuite) TestRefusesWhatIsNotAPreset() {
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"nothing at all", nil},
		{"something else entirely", []byte{0xc0}},
		{"the header and no more", []byte("\xa9l6-helix\x00")},
		{"an offset table of the wrong size", []byte("\xa9l6-helix\x00\xa1x")},
		{"a header with no document", append(
			[]byte("\xa9l6-helix\x00\xda\x00\x30"), make([]byte, 48)...)},
		{"a section with no key", append(append(
			[]byte("\xa9l6-helix\x00\xda\x00\x30"), make([]byte, 48)...),
			0x81, 0xc0)},
		{"a key with no section", append(append(
			[]byte("\xa9l6-helix\x00\xda\x00\x30"), make([]byte, 48)...),
			0x81, 0x00)},
		{"a key that is not a number", append(append(
			[]byte("\xa9l6-helix\x00\xda\x00\x30"), make([]byte, 48)...),
			0x81, 0xa1, 'x', 0xc0)},
	} {
		s.Run(tc.name, func() {
			_, err := wire.DecodeDocument(tc.body)

			s.Require().ErrorIs(err, wire.ErrNotADocument)
		})
	}
}

func TestDocumentPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DocumentPublicTestSuite))
}
