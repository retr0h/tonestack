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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/setlist"
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

func (s *SlotsPublicTestSuite) TestCopyOverwritesTheDestination() {
	doc := s.doc("setlist.hls")

	s.Require().NoError(doc.Copy(
		setlist.Address{Slot: 0}, setlist.Address{Slot: 1}))

	s.Require().Equal("First", doc.Setlists[0].Slots[0].Meta.Name)
	s.Require().Equal("First", doc.Setlists[0].Slots[1].Meta.Name)
}

func (s *SlotsPublicTestSuite) TestSwapExchangesTwoSlots() {
	doc := s.doc("setlist.hls")

	s.Require().NoError(doc.Swap(
		setlist.Address{Slot: 0}, setlist.Address{Slot: 1}))

	s.Require().Equal("Second", doc.Setlists[0].Slots[0].Meta.Name)
	s.Require().Equal("First", doc.Setlists[0].Slots[1].Meta.Name)
}

func (s *SlotsPublicTestSuite) TestSwapUndoesItself() {
	doc := s.doc("setlist.hls")
	before := doc.Setlists[0].Slots[0].Meta.Name

	a, b := setlist.Address{Slot: 0}, setlist.Address{Slot: 3}
	s.Require().NoError(doc.Swap(a, b))
	s.Require().NoError(doc.Swap(a, b))

	s.Require().Equal(before, doc.Setlists[0].Slots[0].Meta.Name)
}

func (s *SlotsPublicTestSuite) TestEditsRejectAnAddressThatIsNotThere() {
	doc := s.doc("setlist.hls")
	good, bad := setlist.Address{Slot: 0}, setlist.Address{Slot: 99}

	tests := []struct {
		name string
		call func() error
	}{
		{"copy from", func() error { return doc.Copy(bad, good) }},
		{"copy to", func() error { return doc.Copy(good, bad) }},
		{"swap first", func() error { return doc.Swap(bad, good) }},
		{"swap second", func() error { return doc.Swap(good, bad) }},
		{"rename", func() error { return doc.Rename(bad, "x") }},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().ErrorIs(tc.call(), setlist.ErrNoSuchSlot)
		})
	}
}

func (s *SlotsPublicTestSuite) TestRename() {
	doc := s.doc("setlist.hls")

	s.Require().NoError(doc.Rename(setlist.Address{Slot: 2}, "Renamed"))

	s.Require().Equal("Renamed", doc.Setlists[0].Slots[2].Meta.Name)
}

func (s *SlotsPublicTestSuite) TestNames() {
	doc := s.doc("setlist.hls")

	got, err := doc.Names(0)

	s.Require().NoError(err)
	s.Require().Equal(
		[]string{"First", "Second", "New Preset", "Unknown Gear"}, got)
}

func (s *SlotsPublicTestSuite) TestNamesRejectsASetlistThatIsNotThere() {
	doc := s.doc("setlist.hls")

	_, err := doc.Names(9)

	s.Require().ErrorIs(err, setlist.ErrNoSuchSlot)
}

func (s *SlotsPublicTestSuite) TestNameIsEmptyWhenMetadataHasNone() {
	sl := setlist.Setlist{Meta: json.RawMessage(`{}`)}

	s.Require().Empty(sl.Name())
}

func (s *SlotsPublicTestSuite) TestWriteRefusesASetlistHoldingSeveral() {
	doc := s.doc("bundle.hlb")
	doc.Schema = setlist.SchemaSetlist

	err := setlist.Write(&bytes.Buffer{}, doc)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "holds one setlist, not 2")
}

func (s *SlotsPublicTestSuite) TestWriteRefusesAPayloadThatCannotEncode() {
	doc := s.doc("setlist.hls")
	// A channel has no JSON representation, so this fails where a real
	// payload never would.
	doc.Setlists[0].Slots[0].Tone = map[string]preset.Tone{
		"dsp0": {"block0": json.RawMessage("not json")},
	}

	err := setlist.Write(&bytes.Buffer{}, doc)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "encoding payload")
}

func (s *SlotsPublicTestSuite) TestWriteReportsAWriterThatFails() {
	err := setlist.Write(&failingWriter{}, s.doc("setlist.hls"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "encoding setlist")
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestSlotsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotsPublicTestSuite))
}
