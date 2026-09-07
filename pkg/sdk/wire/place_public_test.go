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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// PlacePublicTestSuite covers writing a chain into a preset.
//
// Everything here is checked by reading the result back with the decoder the
// rest of this project uses, rather than against the bytes this package
// produced. A block that goes in and does not come out is the failure that
// matters.
type PlacePublicTestSuite struct {
	suite.Suite
}

// amp is a block carrying its own cabinet, the awkward case.
func (s *PlacePublicTestSuite) amp() wire.Placement {
	return wire.Placement{
		Position: 5,
		Model:    6,
		Values:   []any{0.45, 0.6, false, int64(3)},
		Named:    4,
		Enabled:  true,
		Class:    wire.ClassAmpCab,
		Cab:      []any{int64(1), int64(20), int64(12000), 0.0, int64(2), int64(12)},
		CabNamed: 5,
		CabModel: 50,
	}
}

// drive is an ordinary effect.
func (s *PlacePublicTestSuite) drive() wire.Placement {
	return wire.Placement{
		Position: 2,
		Model:    302,
		Values:   []any{0.5, 0.25, true},
		Named:    3,
		Enabled:  false,
		Class:    wire.ClassEffect,
	}
}

func (s *PlacePublicTestSuite) blank() *wire.Document {
	doc, err := wire.Blank()
	s.Require().NoError(err)

	return doc
}

// read runs a document through the decoder the reading commands use.
func (s *PlacePublicTestSuite) read(doc *wire.Document) wire.DevicePreset {
	got, err := wire.DecodePreset(doc.Encode())
	s.Require().NoError(err)

	return got
}

func (s *PlacePublicTestSuite) TestPlace() {
	tests := []struct {
		name   string
		blocks []wire.Placement
		err    error
	}{
		{
			name:   "one effect",
			blocks: []wire.Placement{s.drive()},
		},
		{
			name:   "an amp carrying a cabinet",
			blocks: []wire.Placement{s.amp()},
		},
		{
			name:   "a chain either side of the split",
			blocks: []wire.Placement{s.drive(), s.amp(), s.at(s.drive(), 13)},
		},
		{
			name:   "nothing at all",
			blocks: nil,
		},
		{
			name:   "the last position a block may take",
			blocks: []wire.Placement{s.at(s.drive(), 18)},
		},
		{
			name:   "where the device keeps its input",
			blocks: []wire.Placement{s.at(s.drive(), 0)},
			err:    wire.ErrNoRoom,
		},
		{
			name:   "where the device keeps its split",
			blocks: []wire.Placement{s.at(s.drive(), 9)},
			err:    wire.ErrNoRoom,
		},
		{
			name:   "past the end of the grid",
			blocks: []wire.Placement{s.at(s.drive(), 20)},
			err:    wire.ErrNoRoom,
		},
		{
			name:   "before the start of it",
			blocks: []wire.Placement{s.at(s.drive(), -1)},
			err:    wire.ErrNoRoom,
		},
		{
			name:   "two blocks wanting one position",
			blocks: []wire.Placement{s.drive(), s.at(s.amp(), 2)},
			err:    wire.ErrNoRoom,
		},
		{
			name: "a block with more parameters than a short array counts",
			blocks: []wire.Placement{
				s.wide(s.drive(), 20),
			},
		},
		{
			name: "a value with no encoding",
			blocks: []wire.Placement{
				s.valued(s.drive(), []any{make(chan int)}),
			},
			err: wire.ErrBadValue,
		},
		{
			name: "a cabinet value with no encoding",
			blocks: []wire.Placement{
				s.cabbed(s.amp(), []any{make(chan int)}),
			},
			err: wire.ErrBadValue,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()

			err := wire.Place(doc, tt.blocks)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)

			got := s.read(doc)
			s.Require().Len(got.Blocks, len(tt.blocks))

			for i, want := range tt.blocks {
				s.Require().Equal(want.Position, got.Blocks[i].Index)
				s.Require().Equal(want.Model, got.Blocks[i].Model)
				s.Require().Equal(want.Enabled, got.Blocks[i].Enabled)
				s.Require().Equal(s.asDevice(want.Values), got.Blocks[i].Values)
				s.Require().Equal(s.asDevice(want.Cab), got.Blocks[i].Cab)
				s.Require().Equal(want.CabNamed, got.Blocks[i].CabNamed)
			}
		})
	}
}

// TestPlaceLeavesTheRoutingAlone is why a preset is written into rather than
// built. The input, split, join and output are the device's own and nothing
// here understands them well enough to write one.
func (s *PlacePublicTestSuite) TestPlaceLeavesTheRoutingAlone() {
	doc := s.blank()
	before := s.read(doc).Routing

	s.Require().NoError(wire.Place(doc, []wire.Placement{s.amp(), s.drive()}))

	s.Require().Equal(before, s.read(doc).Routing)
	s.Require().NotEmpty(before)
}

// TestPlaceWritesEverySnapshot is the coupling that would otherwise be
// missed. Each snapshot holds a record of which grid position is switched
// on, so a chain written without them recalls the wrong blocks.
func (s *PlacePublicTestSuite) TestPlaceWritesEverySnapshot() {
	doc := s.blank()

	s.Require().NoError(wire.Place(doc, []wire.Placement{s.amp(), s.drive()}))

	for snap := range s.read(doc).Snapshots {
		for _, b := range []wire.Placement{s.amp(), s.drive()} {
			path := wire.Path{10, snap, 3, b.Position, 1}
			body, ok := doc.Section(10)
			s.Require().True(ok)

			start, end, err := wire.Locate(body, path)
			s.Require().NoError(err, "snapshot %d position %d", snap, b.Position)
			s.Require().Equal(s.messagePackBool(b.Enabled), []byte(body[start:end]),
				"snapshot %d disagrees about position %d", snap, b.Position)
		}
	}
}

// TestPlaceEmptiesWhatTheChainDoesNotName keeps a preset saying one thing.
func (s *PlacePublicTestSuite) TestPlaceEmptiesWhatTheChainDoesNotName() {
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
	s.Require().NoError(err)

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)
	s.Require().Len(s.read(doc).Blocks, 6, "the capture has six blocks")

	s.Require().NoError(wire.Place(doc, []wire.Placement{s.drive()}))

	got := s.read(doc)
	s.Require().Len(got.Blocks, 1)
	s.Require().Equal(302, got.Blocks[0].Model)
}

// TestPlaceReportsADocumentWithNoChain covers bytes no device would send.
func (s *PlacePublicTestSuite) TestPlaceReportsADocumentWithNoChain() {
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
	s.Require().NoError(err)

	full, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	s.Require().ErrorIs(
		wire.Place(wire.NewDocument(full, []int8{1}), nil), wire.ErrNotADocument)
}

// TestPlaceWithoutSnapshots covers a preset carrying no snapshot section.
func (s *PlacePublicTestSuite) TestPlaceWithoutSnapshots() {
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
	s.Require().NoError(err)

	full, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	doc := wire.NewDocument(full, []int8{0})
	s.Require().NoError(wire.Place(doc, []wire.Placement{s.drive()}))
}

// TestBlankIsAPresetADeviceWrote checks the embedded copy every call starts
// from. A build shipping a broken one should say so rather than write half a
// preset to somebody's hardware.
func (s *PlacePublicTestSuite) TestBlankIsAPresetADeviceWrote() {
	doc, err := wire.Blank()
	s.Require().NoError(err)

	got := s.read(doc)
	s.Require().Empty(got.Blocks, "an unused slot holds no blocks")
	s.Require().Len(got.Snapshots, 3, "a device ships three snapshots")
	s.Require().NotEmpty(got.Routing, "and the routing a chain sits in")

	other, err := wire.Blank()
	s.Require().NoError(err)
	s.Require().NoError(wire.Place(other, []wire.Placement{s.drive()}))
	s.Require().Empty(s.read(doc).Blocks, "each call gets its own document")
}

// TestOpen covers which grid positions a chain may use.
func (s *PlacePublicTestSuite) TestOpen() {
	got, err := wire.Open(s.blank())

	s.Require().NoError(err)
	s.Require().Equal(
		[]int{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 13, 14, 15, 16, 17, 18}, got,
		"the input, split, join and output take the other four")
}

// TestOpenOnADocumentWithNoChain covers bytes no device would send.
func (s *PlacePublicTestSuite) TestOpenOnADocumentWithNoChain() {
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
	s.Require().NoError(err)

	full, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	_, err = wire.Open(wire.NewDocument(full, []int8{1}))
	s.Require().ErrorIs(err, wire.ErrNotADocument)
}

// TestPlaceInOrder covers fitting a chain into the positions a device has
// free, which is what an import does with one.
func (s *PlacePublicTestSuite) TestPlaceInOrder() {
	tests := []struct {
		name   string
		blocks []wire.Placement
		want   []int
		err    error
	}{
		{
			name: "a chain out of order keeps its order and moves up",
			blocks: []wire.Placement{
				s.at(s.drive(), 9), s.at(s.amp(), 4), s.at(s.drive(), 6),
			},
			want: []int{1, 2, 3},
		},
		{
			name:   "a chain already where the device would put it",
			blocks: []wire.Placement{s.at(s.drive(), 1), s.at(s.amp(), 2)},
			want:   []int{1, 2},
		},
		{
			name:   "nothing at all",
			blocks: nil,
			want:   []int{},
		},
		{
			name:   "more blocks than the device lays out",
			blocks: make([]wire.Placement, wire.GridSize+1),
			err:    wire.ErrNoRoom,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()

			err := wire.PlaceInOrder(doc, tt.blocks)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)
				s.Require().Contains(err.Error(), "lays out")

				return
			}

			s.Require().NoError(err)

			got := []int{}
			for _, b := range tt.blocks {
				got = append(got, b.Position)
			}

			s.Require().Equal(tt.want, got)
			s.Require().Len(s.read(doc).Blocks, len(tt.blocks))
		})
	}
}

// TestPlaceInOrderOnADocumentWithNoChain covers bytes no device would send.
func (s *PlacePublicTestSuite) TestPlaceInOrderOnADocumentWithNoChain() {
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
	s.Require().NoError(err)

	full, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	s.Require().ErrorIs(
		wire.PlaceInOrder(wire.NewDocument(full, []int8{1}), nil),
		wire.ErrNotADocument)
}

// TestNoRoomErrorNamesThePosition covers what a caller reads.
func (s *PlacePublicTestSuite) TestNoRoomErrorNamesThePosition() {
	err := &wire.NoRoomError{Position: 9, Why: "the device keeps its routing there"}

	s.Require().Contains(err.Error(), "position 9")
	s.Require().Contains(err.Error(), "routing")
	s.Require().ErrorIs(err, wire.ErrNoRoom)
}

// TestPlaceOnAChainTheDeviceDidNotWrite covers grids no device produces.
//
// Every one of these is reached by rebuilding the chain rather than by
// asking a capture for something it does not have.
func (s *PlacePublicTestSuite) TestPlaceOnAChainTheDeviceDidNotWrite() {
	tests := []struct {
		name  string
		chain []byte
		want  int
	}{
		{
			name:  "a grid shorter than the device lays out",
			chain: []byte{0x91, 0x82, 0x13, 0x08, 0x14, 0xc0},
			want:  0,
		},
		{
			name:  "a position that is not a map",
			chain: []byte{0x91, 0x2a},
			want:  0,
		},
		{
			name:  "a position not saying what kind it is",
			chain: []byte{0x91, 0x81, 0x14, 0xc0},
			want:  0,
		},
		{
			name:  "a kind that is not a number",
			chain: []byte{0x91, 0x82, 0x13, 0xa1, 0x61, 0x14, 0xc0},
			want:  0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()

			body, ok := doc.Section(0)
			s.Require().True(ok)

			body, err := wire.SpliceRaw(body, wire.Path{22}, tt.chain)
			s.Require().NoError(err)
			doc.SetSection(0, body)

			s.Require().NoError(wire.Place(doc, nil))
			s.Require().Len(s.read(doc).Blocks, tt.want)
		})
	}
}

// TestPlaceOnSnapshotsTheDeviceDidNotWrite covers the same for section 10.
func (s *PlacePublicTestSuite) TestPlaceOnSnapshotsTheDeviceDidNotWrite() {
	tests := []struct {
		name    string
		snaps   []byte
		section []byte
		err     error
	}{
		{
			name:    "a section with no snapshot list at all",
			section: []byte{0x81, 0x06, 0x00},
			err:     wire.ErrNoSuchPath,
		},
		{
			name:  "a snapshot list that is not a list",
			snaps: []byte{0x2a},
			err:   wire.ErrNoSuchPath,
		},
		{
			name:  "a snapshot with no record of the grid",
			snaps: []byte{0x91, 0x80},
		},
		{
			name:  "a snapshot keeping a shorter record than the grid",
			snaps: []byte{0x91, 0x81, 0x03, 0x91, 0x92, 0xc2, 0xc2},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()

			if tt.section != nil {
				doc.SetSection(10, tt.section)
			} else {
				body, ok := doc.Section(10)
				s.Require().True(ok)

				body, err := wire.SpliceRaw(body, wire.Path{10}, tt.snaps)
				s.Require().NoError(err)
				doc.SetSection(10, body)
			}

			err := wire.Place(doc, []wire.Placement{s.drive()})

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(s.read(doc).Blocks, 1)
		})
	}
}

// wide gives a placement more parameters than a small array header counts.
func (s *PlacePublicTestSuite) wide(
	b wire.Placement,
	n int,
) wire.Placement {
	b.Values = make([]any, n)
	for i := range b.Values {
		b.Values[i] = float64(i) / float64(n)
	}

	b.Named = n

	return b
}

// valued replaces a placement's parameters.
func (s *PlacePublicTestSuite) valued(
	b wire.Placement,
	values []any,
) wire.Placement {
	b.Values = values

	return b
}

// cabbed replaces a placement's cabinet values.
func (s *PlacePublicTestSuite) cabbed(
	b wire.Placement,
	values []any,
) wire.Placement {
	b.Cab = values

	return b
}

// at moves a placement to another position.
func (s *PlacePublicTestSuite) at(
	b wire.Placement,
	position int,
) wire.Placement {
	b.Position = position

	return b
}

// asDevice is what a parameter reads as once a device has held it.
//
// Every float in a preset is a float32. There are 140 of them across the
// captures and not one float64, so 0.45 written to a device reads back as
// 0.44999998807907104. A rig read off hardware already carries values that
// have been through this and so survives it unchanged; one somebody typed
// gets rounded to what the device can store.
func (s *PlacePublicTestSuite) asDevice(values []any) []any {
	if values == nil {
		return nil
	}

	out := make([]any, 0, len(values))

	for _, v := range values {
		if f, ok := v.(float64); ok {
			out = append(out, float64(float32(f)))

			continue
		}

		out = append(out, v)
	}

	return out
}

// messagePackBool is the one byte the format spends on a bool.
func (s *PlacePublicTestSuite) messagePackBool(v bool) []byte {
	if v {
		return []byte{0xc3}
	}

	return []byte{0xc2}
}

func TestPlacePublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlacePublicTestSuite))
}
