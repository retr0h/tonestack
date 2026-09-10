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

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
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

// capture returns a document a device wrote, optionally rebuilt holding only
// the given sections.
func (s *PlacePublicTestSuite) capture(sections ...int8) *wire.Document {
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
	s.Require().NoError(err)

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	if len(sections) == 0 {
		return doc
	}

	return wire.NewDocument(doc, sections)
}

// TestPlace writes a chain into a preset.
func (s *PlacePublicTestSuite) TestPlace() {
	tests := []struct {
		name   string
		blocks []wire.Placement
		// which document to write into: a blank slot unless a case says
		// otherwise.
		doc string
		// the routing must survive untouched.
		routing bool
		// every snapshot must agree about every block.
		snapshots bool
		// and must leave the positions a chain cannot take exactly as the
		// device wrote them.
		keepsRouting bool
		err          error
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
			name: "nothing at all",
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
			name:   "a block with more parameters than a short array counts",
			blocks: []wire.Placement{s.wide(s.drive(), 20)},
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
		{
			// Whatever the slot held is gone, which keeps a preset saying one
			// thing.
			name:   "a slot already holding six blocks",
			blocks: []wire.Placement{s.drive()},
			doc:    "capture",
		},
		{
			name:   "a preset carrying no snapshot section",
			blocks: []wire.Placement{s.drive()},
			doc:    "snapshotless",
		},
		{
			name: "a document with no chain in it, which no device would send",
			doc:  "chainless",
			err:  wire.ErrNotADocument,
		},
		{
			// This is why a preset is written into rather than built. The
			// input, split, join and output are the device's own and nothing
			// here understands them well enough to write one.
			name:    "the routing, which the device keeps to itself",
			blocks:  []wire.Placement{s.drive(), s.amp()},
			routing: true,
		},
		{
			// The coupling that would otherwise be missed. Each snapshot
			// holds a record of which grid position is switched on, so a
			// chain written without them recalls the wrong blocks.
			name:      "every snapshot, and not only the chain",
			blocks:    []wire.Placement{s.drive(), s.amp()},
			snapshots: true,
		},
		{
			// The device keeps its input, split, join and output on the same
			// grid, and a snapshot records those too. Writing them from a
			// chain that never mentions them switched the split and the join
			// off, so recalling the snapshot bypassed the routing.
			name:         "the routing, inside every snapshot",
			blocks:       []wire.Placement{s.drive()},
			keepsRouting: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()

			switch tt.doc {
			case "capture":
				doc = s.capture()
				s.Require().Len(s.read(doc).Blocks, 6, "the capture has six blocks")
			case "chainless":
				doc = s.capture(1)
			case "snapshotless":
				doc = s.capture(0)
			}

			before := s.read(doc).Routing

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

			if tt.routing {
				s.Require().NotEmpty(before)
				s.Require().Equal(before, got.Routing)
			}

			if tt.keepsRouting {
				s.keepsRouting(doc)

				return
			}

			if !tt.snapshots {
				return
			}

			body, ok := doc.Section(10)
			s.Require().True(ok)

			for snap := range got.Snapshots {
				for _, b := range tt.blocks {
					start, end, err := wire.Locate(body, wire.Path{10, snap, 3, b.Position, 1})
					s.Require().NoError(err, "snapshot %d position %d", snap, b.Position)
					s.Require().Equal(s.messagePackBool(b.Enabled), []byte(body[start:end]),
						"snapshot %d disagrees about position %d", snap, b.Position)
				}
			}
		})
	}
}

// keepsRouting checks that every position a chain cannot take still holds the
// byte the device wrote, in every snapshot.
func (s *PlacePublicTestSuite) keepsRouting(doc *wire.Document) {
	fresh := s.blank()

	open, err := wire.Open(fresh)
	s.Require().NoError(err)

	takeable := make(map[int]bool, len(open))
	for _, p := range open {
		takeable[p] = true
	}

	was, ok := fresh.Section(10)
	s.Require().True(ok)

	now, ok := doc.Section(10)
	s.Require().True(ok)

	for snap := range s.read(doc).Snapshots {
		for i := range wire.GridSize {
			if takeable[i] {
				continue
			}

			path := wire.Path{10, snap, 3, i, 1}

			from, to, err := wire.Locate(was, path)
			if err != nil {
				continue
			}

			at, end, err := wire.Locate(now, path)
			s.Require().NoError(err)

			s.Require().Equal(was[from:to], now[at:end],
				"snapshot %d rewrote position %d, which the device owns", snap, i)
		}
	}
}

// TestBlank checks the embedded copy every call starts from. A build shipping
// a broken one should say so rather than write half a preset to somebody's
// hardware.
func (s *PlacePublicTestSuite) TestBlank() {
	doc := s.blank()

	got := s.read(doc)
	s.Require().Empty(got.Blocks, "an unused slot holds no blocks")
	s.Require().Len(got.Snapshots, 3, "a device ships three snapshots")
	s.Require().NotEmpty(got.Routing, "and the routing a chain sits in")

	s.Require().NoError(wire.Place(s.blank(), []wire.Placement{s.drive()}))
	s.Require().Empty(s.read(doc).Blocks, "each call gets its own document")
}

// TestOpen covers which grid positions a chain may use.
func (s *PlacePublicTestSuite) TestOpen() {
	tests := []struct {
		name      string
		chainless bool
		want      []int
		err       error
	}{
		{
			name: "an unused slot",
			// The input, split, join and output take the other four.
			want: []int{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 13, 14, 15, 16, 17, 18},
		},
		{
			name:      "a document with no chain, which no device would send",
			chainless: true,
			err:       wire.ErrNotADocument,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()
			if tt.chainless {
				doc = s.capture(1)
			}

			got, err := wire.Open(doc)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestPlaceAsWritten covers putting a chain where the preset says it goes.
//
// The two numberings differ by one, measured against slot 27B of an HX
// Stomp: the preset HX Edit exported puts its six blocks at 1 through 6 and
// the document the device sent puts the same six at 2 through 7.
func (s *PlacePublicTestSuite) TestPlaceAsWritten() {
	tests := []struct {
		name      string
		blocks    []wire.Placement
		chainless bool
		want      []int
		err       error
	}{
		{
			name: "the six blocks of the bass preset",
			blocks: []wire.Placement{
				s.at(s.drive(), 1), s.at(s.drive(), 2), s.at(s.drive(), 3),
				s.at(s.drive(), 4), s.at(s.amp(), 5), s.at(s.drive(), 6),
			},
			want: []int{2, 3, 4, 5, 6, 7},
		},
		{
			name:   "one block at the start of a path",
			blocks: []wire.Placement{s.at(s.drive(), 0)},
			want:   []int{1},
		},
		{
			name: "nothing at all",
			want: []int{},
		},
		{
			name:   "a position the device keeps its split on",
			blocks: []wire.Placement{s.at(s.drive(), 8)},
			err:    wire.ErrNoRoom,
		},
		{
			name:   "a position past the end of the grid",
			blocks: []wire.Placement{s.at(s.drive(), wire.GridSize)},
			err:    wire.ErrNoRoom,
		},
		{
			name:      "a document with no chain, which no device would send",
			chainless: true,
			err:       wire.ErrNotADocument,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := s.blank()
			if tt.chainless {
				doc = s.capture(1)
			}

			err := wire.PlaceAsWritten(doc, tt.blocks)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)

			// Where the blocks landed, read back out of the document. The
			// chain handed over is the caller's and comes back unshifted.
			read := s.read(doc)

			got := []int{}
			for _, b := range read.Blocks {
				got = append(got, b.Index)
			}

			s.Require().Equal(tt.want, got)
			s.Require().Len(read.Blocks, len(tt.blocks))

			for i, b := range tt.blocks {
				s.Require().Equal(tt.want[i]-wire.GridOffset, b.Position,
					"the caller's chain must come back as it went in")
			}
		})
	}
}

// TestNoRoomError covers what a caller reads.
func (s *PlacePublicTestSuite) TestNoRoomError() {
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
