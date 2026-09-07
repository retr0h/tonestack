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

package wire

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"
)

// PresetShapeTestSuite covers a device answering with the wrong shape.
//
// Everything here comes off a wire, so nothing about it is guaranteed. A
// malformed answer must produce an empty reading rather than a panic, because
// the alternative is a crash mid-session on hardware somebody is playing.
type PresetShapeTestSuite struct {
	suite.Suite
}

// encode renders a document the way a device would.
func (s *PresetShapeTestSuite) encode(doc any) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeString(magic))
	s.Require().NoError(enc.EncodeString("offsets"))
	s.Require().NoError(enc.Encode(doc))

	return buf.Bytes()
}

func (s *PresetShapeTestSuite) TestADocumentSayingNothingUseful() {
	for _, tc := range []struct {
		name string
		doc  map[int8]any
	}{
		{"no tone at all", map[int8]any{}},
		{"a tone that is not a map", map[int8]any{keyTone: "text"}},
		{"blocks that are not a list", map[int8]any{
			keyTone: map[int8]any{keyBlocks: "text"},
		}},
		{"an entry that is not a map", map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{"text"}},
		}},
		{"an entry of another kind", map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: 8},
			}},
		}},
		{"a body that is not a map", map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: kindBlock, keyBlockBody: "text"},
			}},
		}},
		{"a block naming no model", map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: kindBlock, keyBlockBody: map[int8]any{}},
			}},
		}},
		{"a model reference that is not a number", map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{
					keyBlockKind: kindBlock,
					keyBlockBody: map[int8]any{
						keyModelRef: map[int8]any{keyModelNum: "text"},
					},
				},
			}},
		}},
	} {
		s.Run(tc.name, func() {
			got, err := DecodePreset(s.encode(tc.doc))

			s.Require().NoError(err)
			s.Require().Empty(got.Blocks)
		})
	}
}

func (s *PresetShapeTestSuite) TestSnapshotsInTheWrongShape() {
	for _, doc := range []map[int8]any{
		{keySnapshots: "text"},
		{keySnapshots: map[int8]any{keySnapList: "text"}},
		{keySnapshots: map[int8]any{keySnapList: []any{"text"}}},
	} {
		got, err := DecodePreset(s.encode(doc))

		s.Require().NoError(err)
		s.Require().Empty(got.Snapshots)
	}
}

func (s *PresetShapeTestSuite) TestFootswitchesInTheWrongShape() {
	for _, doc := range []map[int8]any{
		{keyFootswitch: "text"},
		{keyFootswitch: map[int8]any{keyFsPaths: "text"}},
		{keyFootswitch: map[int8]any{keyFsPaths: []any{"text"}}},
		{keyFootswitch: map[int8]any{keyFsPaths: []any{[]any{"text"}}}},
		{keyFootswitch: map[int8]any{keyFsPaths: []any{[]any{
			map[int8]any{keyFsBody: "text"},
		}}}},
		{keyFootswitch: map[int8]any{keyFsPaths: []any{[]any{
			map[int8]any{keyFsBody: map[int8]any{keyFsModel: ""}},
		}}}},
	} {
		got, err := DecodePreset(s.encode(doc))

		s.Require().NoError(err)
		s.Require().Empty(got.Footswitches)
	}
}

func (s *PresetShapeTestSuite) TestValuesOfEveryWidth() {
	// MessagePack carries a number in whichever width holds it, so the same
	// parameter arrives differently from one preset to the next.
	got := narrow([]any{
		true, float32(0.5), 1.5, int8(1), int16(2), int32(3), int64(4),
		uint8(5), uint16(6), uint32(7), uint64(8), "text",
	})

	s.Require().Equal([]any{
		true, 0.5, 1.5,
		int64(1), int64(2), int64(3), int64(4),
		int64(5), int64(6), int64(7), int64(8),
		"text",
	}, got)
}

func (s *PresetShapeTestSuite) TestASwitchShowsWhatThePedalPrints() {
	// Somebody's own words when they set them, and the block's name when they
	// did not. A device carries both and flags which it is showing.
	for _, tc := range []struct {
		name  string
		entry map[int8]any
		want  string
	}{
		{"a label somebody set", map[int8]any{
			keyFsNamed: true, keyFsLabel: "60s / 70s\x00",
			keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
		}, "60s / 70s"},
		{"no label set", map[int8]any{
			keyFsNamed: false, keyFsLabel: "ignored\x00",
			keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
		}, "Ampeg B-15NF"},
		{"a label flagged but empty", map[int8]any{
			keyFsNamed: true, keyFsLabel: "\x00",
			keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
		}, "Ampeg B-15NF"},
		{"a label flagged but of the wrong kind", map[int8]any{
			keyFsNamed: true, keyFsLabel: 7,
			keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
		}, "Ampeg B-15NF"},
	} {
		s.Run(tc.name, func() {
			got, err := DecodePreset(s.encode(map[int8]any{
				keyFootswitch: map[int8]any{keyFsPaths: []any{[]any{tc.entry}}},
			}))

			s.Require().NoError(err)
			s.Require().Len(got.Footswitches, 1)
			s.Require().Equal(tc.want, got.Footswitches[0].Label)
			s.Require().Equal("Ampeg B-15NF", got.Footswitches[0].Gear)
		})
	}
}

func (s *PresetShapeTestSuite) TestReadsANumberOfAnyWidth() {
	// A tempo arrives as a float in one preset and as an integer in another,
	// because MessagePack carries a value in the narrowest form that fits.
	for _, tc := range []struct {
		in   any
		want float64
	}{
		{1.5, 1.5},
		{float32(2.5), 2.5},
		{int8(3), 3},
		{int64(4), 4},
		{uint8(5), 5},
		{uint64(6), 6},
	} {
		got, ok := asFloat(tc.in)

		s.Require().True(ok)
		s.Require().InDelta(tc.want, got, 0.001)
	}

	_, ok := asFloat("text")
	s.Require().False(ok)
}

func TestPresetShapeTestSuite(t *testing.T) {
	suite.Run(t, new(PresetShapeTestSuite))
}
