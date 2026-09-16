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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// SnapshotsPublicTestSuite covers writing somebody else's snapshots into a
// preset.
//
// Checked by reading the result back with the decoder the rest of this project
// uses, rather than against the bytes this package produced. A snapshot that
// goes in and does not come out is the failure that matters.
type SnapshotsPublicTestSuite struct {
	suite.Suite
}

func (s *SnapshotsPublicTestSuite) blank() *wire.Document {
	doc, err := wire.Blank()
	s.Require().NoError(err)

	return doc
}

// read decodes a document the way a device's answer is read.
func (s *SnapshotsPublicTestSuite) read(
	doc *wire.Document,
) wire.DevicePreset {
	got, err := wire.DecodePreset(doc.Encode())
	s.Require().NoError(err)

	return got
}

// snapshot states every field, which is the case that exercises all of them.
func (s *SnapshotsPublicTestSuite) snapshot(
	name string,
	tempo float64,
	led int,
	valid bool,
	on map[int]bool,
) wire.Snapshot {
	return wire.Snapshot{
		Name: &name, Tempo: &tempo, LED: &led, Valid: &valid, On: on,
	}
}

// TestEverythingASnapshotSays writes all four fields and reads them back.
func (s *SnapshotsPublicTestSuite) TestEverythingASnapshotSays() {
	doc := s.blank()

	wire.PlaceSnapshots(doc, []wire.Snapshot{
		s.snapshot("Verse", 92, 3, true, map[int]bool{5: true, 6: false}),
		s.snapshot("Chorus", 140.5, 1, false, map[int]bool{5: false, 6: true}),
	})

	got := s.read(doc)
	s.Require().Len(got.Snapshots, 3, "the blank's three, none added")

	first, second := got.Snapshots[0], got.Snapshots[1]

	s.Require().Equal("Verse", first.Name)
	s.Require().InDelta(92, first.Tempo, 0.001)
	s.Require().Equal(3, first.LED)
	s.Require().True(first.Valid)
	s.Require().True(first.On[5])
	s.Require().False(first.On[6])

	s.Require().Equal("Chorus", second.Name)
	s.Require().InDelta(140.5, second.Tempo, 0.001)
	s.Require().Equal(1, second.LED)
	s.Require().False(second.Valid)
	s.Require().False(second.On[5])
	s.Require().True(second.On[6])
}

// TestThreeSnapshotsRecallThreeSounds is the whole point of a snapshot.
//
// Before this, every snapshot took its record from the chain, so all three
// switched the same blocks and a preset holding three sounds arrived with one.
func (s *SnapshotsPublicTestSuite) TestThreeSnapshotsRecallThreeSounds() {
	doc := s.blank()

	wire.PlaceSnapshots(doc, []wire.Snapshot{
		{On: map[int]bool{4: true, 5: true}},
		{On: map[int]bool{4: false, 5: true}},
		{On: map[int]bool{4: false, 5: false}},
	})

	got := s.read(doc)

	s.Require().True(got.Snapshots[0].On[4])
	s.Require().False(got.Snapshots[1].On[4])
	s.Require().True(got.Snapshots[1].On[5])
	s.Require().False(got.Snapshots[2].On[5])
}

// TestTheRoutingIsLeftAlone covers a position the chain may not take.
//
// A device lays its input, its split, its join and its output on the same grid
// a chain sits on: the first, the two in the middle and the last. A caller
// asking for one of those to go off is asking for a snapshot that recalls the
// chain with its routing bypassed, so it is refused by being ignored.
func (s *SnapshotsPublicTestSuite) TestTheRoutingIsLeftAlone() {
	// One of the two in the middle. Whatever it holds is the device's, and the
	// opposite is what gets asked for, so this holds whichever way the blank
	// was written.
	const split = 9

	doc := s.blank()

	before := s.read(doc)
	held := before.Snapshots[0].On[split]

	wire.PlaceSnapshots(doc, []wire.Snapshot{
		{On: map[int]bool{split: !held, 5: false}},
	})

	got := s.read(doc)

	s.Require().Equal(held, got.Snapshots[0].On[split],
		"the routing keeps what the device gave it")
	s.Require().False(got.Snapshots[0].On[5], "and a block position took")
}

// TestAPositionNobodyNames leaves what the preset already held.
func (s *SnapshotsPublicTestSuite) TestAPositionNobodyNames() {
	doc := s.blank()

	before := s.read(doc)
	held := before.Snapshots[0].On[7]

	wire.PlaceSnapshots(doc, []wire.Snapshot{{On: map[int]bool{5: true}}})

	got := s.read(doc)
	s.Require().Equal(held, got.Snapshots[0].On[7])
}

// TestMoreSnapshotsThanThePresetHolds writes the ones it can.
func (s *SnapshotsPublicTestSuite) TestMoreSnapshotsThanThePresetHolds() {
	doc := s.blank()

	name := "Fifth"
	wire.PlaceSnapshots(doc, []wire.Snapshot{{}, {}, {}, {}, {Name: &name}})

	got := s.read(doc)
	s.Require().Len(got.Snapshots, 3)
	s.Require().Equal("SNAPSHOT 3", got.Snapshots[2].Name)
}

// TestNoSnapshots changes nothing.
func (s *SnapshotsPublicTestSuite) TestNoSnapshots() {
	doc := s.blank()
	before := doc.Encode()

	wire.PlaceSnapshots(doc, nil)

	s.Require().Equal(before, doc.Encode())
}

// TestAPresetKeepingNoSnapshots has none to overwrite.
func (s *SnapshotsPublicTestSuite) TestAPresetKeepingNoSnapshots() {
	doc := wire.NewDocument(s.blank(), []int8{wire.KeyTone})
	before := doc.Encode()

	name := "Verse"
	wire.PlaceSnapshots(doc, []wire.Snapshot{{Name: &name}})

	s.Require().Equal(before, doc.Encode())
}

// TestAPresetWithNoChain is left alone.
//
// Which positions hold the routing is only knowable from the chain, and a
// snapshot written without knowing would switch the routing off. Writing a
// chain refuses such a document first, so nothing is lost by saying nothing.
func (s *SnapshotsPublicTestSuite) TestAPresetWithNoChain() {
	doc := wire.NewDocument(s.blank(), []int8{wire.KeySnapshots})
	before := doc.Encode()

	name := "Verse"
	wire.PlaceSnapshots(doc, []wire.Snapshot{{Name: &name}})

	s.Require().Equal(before, doc.Encode())
}

func TestSnapshotsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SnapshotsPublicTestSuite))
}
