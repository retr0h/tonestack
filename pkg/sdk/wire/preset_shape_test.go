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

// footswitch wraps one switch entry the way a preset carries it.
func footswitch(entry map[int8]any) map[int8]any {
	return map[int8]any{
		keyFootswitch: map[int8]any{keyFsPaths: []any{[]any{entry}}},
	}
}

// TestDecodePreset reads a preset a device sent, however it is shaped.
func (s *PresetShapeTestSuite) TestDecodePreset() {
	tests := []struct {
		name string
		doc  map[int8]any
		// routing entries the reading must hold, none unless a case says so.
		routing int
		// the label a single footswitch must show, when one is expected.
		label string
	}{
		{name: "no tone at all", doc: map[int8]any{}},
		{name: "a tone that is not a map", doc: map[int8]any{keyTone: "text"}},
		{name: "blocks that are not a list", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: "text"},
		}},
		{name: "a block that is not a map", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{"text"}},
		}},
		{name: "a block of another kind", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: 8},
			}},
		}},
		{name: "a block body that is not a map", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: kindBlock, keyBlockBody: "text"},
			}},
		}},
		{name: "a block naming no model", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: kindBlock, keyBlockBody: map[int8]any{}},
			}},
		}},
		{name: "a model reference that is not a number", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{
					keyBlockKind: kindBlock,
					keyBlockBody: map[int8]any{
						keyModelRef: map[int8]any{keyModelNum: "text"},
					},
				},
			}},
		}},
		{name: "snapshots that are not a map", doc: map[int8]any{
			keySnapshots: "text",
		}},
		{name: "a snapshot list that is not a list", doc: map[int8]any{
			keySnapshots: map[int8]any{keySnapList: "text"},
		}},
		{name: "a snapshot that is not a map", doc: map[int8]any{
			keySnapshots: map[int8]any{keySnapList: []any{"text"}},
		}},
		{name: "footswitches that are not a map", doc: map[int8]any{
			keyFootswitch: "text",
		}},
		{name: "switch paths that are not a list", doc: map[int8]any{
			keyFootswitch: map[int8]any{keyFsPaths: "text"},
		}},
		{name: "a switch path that is not a list", doc: map[int8]any{
			keyFootswitch: map[int8]any{keyFsPaths: []any{"text"}},
		}},
		{name: "a switch that is not a map", doc: map[int8]any{
			keyFootswitch: map[int8]any{keyFsPaths: []any{[]any{"text"}}},
		}},
		{
			name: "a switch body that is not a map",
			doc:  footswitch(map[int8]any{keyFsBody: "text"}),
		},
		{
			name: "a switch naming nothing",
			doc: footswitch(map[int8]any{
				keyFsBody: map[int8]any{keyFsModel: ""},
			}),
		},
		// Somebody's own words when they set them, and the block's name when
		// they did not. A device carries both and flags which it is showing.
		{
			name: "a label somebody set",
			doc: footswitch(map[int8]any{
				keyFsNamed: true, keyFsLabel: "60s / 70s\x00",
				keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
			}),
			label: "60s / 70s",
		},
		{
			name: "no label set",
			doc: footswitch(map[int8]any{
				keyFsNamed: false, keyFsLabel: "ignored\x00",
				keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
			}),
			label: "Ampeg B-15NF",
		},
		{
			name: "a label flagged but empty",
			doc: footswitch(map[int8]any{
				keyFsNamed: true, keyFsLabel: "\x00",
				keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
			}),
			label: "Ampeg B-15NF",
		},
		{
			name: "a label flagged but of the wrong kind",
			doc: footswitch(map[int8]any{
				keyFsNamed: true, keyFsLabel: 7,
				keyFsBody: map[int8]any{keyFsModel: "Ampeg B-15NF\x00"},
			}),
			label: "Ampeg B-15NF",
		},
		{name: "a routing kind that is not a number", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{map[int8]any{
				keyBlockKind: "text", keyBlockBody: map[int8]any{},
			}}},
		}},
		{name: "a routing body that is not a map", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{map[int8]any{
				keyBlockKind: kindInput, keyBlockBody: "text",
			}}},
		}},
		{name: "a split with nothing after it", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{map[int8]any{
				keyBlockKind: kindSplit,
				keyBlockBody: map[int8]any{keySplitBlock: "text"},
			}}},
		}},
		{name: "a join with nothing before it", doc: map[int8]any{
			keyTone: map[int8]any{keyBlocks: []any{map[int8]any{
				keyBlockKind: kindJoin,
				keyBlockBody: map[int8]any{keyJoinBlock: "text"},
			}}},
		}},
		// A routing entry that names itself but says nothing readable about
		// its settings is still a routing entry.
		{
			name: "settings that are not a map",
			doc: map[int8]any{keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: kindInput, keyBlockBody: map[int8]any{
					keyInputSelect: 1, keyFlowParams: "text",
				}},
			}}},
			routing: 1,
		},
		{
			name: "settings holding values that are not a map",
			doc: map[int8]any{keyTone: map[int8]any{keyBlocks: []any{
				map[int8]any{keyBlockKind: kindInput, keyBlockBody: map[int8]any{
					keyInputSelect: 1,
					keyFlowParams:  map[int8]any{keyValues: "text"},
				}},
			}}},
			routing: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := DecodePreset(s.encode(tt.doc))

			s.Require().NoError(err)
			s.Require().Empty(got.Blocks)
			s.Require().Empty(got.Snapshots)
			s.Require().Len(got.Routing, tt.routing)

			for _, r := range got.Routing {
				s.Require().Empty(r.Values)
			}

			if tt.label == "" {
				s.Require().Empty(got.Footswitches)

				return
			}

			s.Require().Len(got.Footswitches, 1)
			s.Require().Equal(tt.label, got.Footswitches[0].Label)
			s.Require().Equal("Ampeg B-15NF", got.Footswitches[0].Gear)
		})
	}
}

// TestNarrow reads a value in whichever width it arrived in. MessagePack
// carries a number in the narrowest form that holds it, so the same parameter
// arrives differently from one preset to the next.
func (s *PresetShapeTestSuite) TestNarrow() {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{name: "a flag", in: true, want: true},
		{name: "a single-width float", in: float32(0.5), want: 0.5},
		{name: "a double-width float", in: 1.5, want: 1.5},
		{name: "a one-byte integer", in: int8(1), want: int64(1)},
		{name: "a two-byte integer", in: int16(2), want: int64(2)},
		{name: "a four-byte integer", in: int32(3), want: int64(3)},
		{name: "an eight-byte integer", in: int64(4), want: int64(4)},
		{name: "an unsigned byte", in: uint8(5), want: int64(5)},
		{name: "two unsigned bytes", in: uint16(6), want: int64(6)},
		{name: "four unsigned bytes", in: uint32(7), want: int64(7)},
		{name: "eight unsigned bytes", in: uint64(8), want: int64(8)},
		{name: "something that is not a number", in: "text", want: "text"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal([]any{tt.want}, narrow([]any{tt.in}))
		})
	}
}

// TestAsFloat reads a number of any width. A tempo arrives as a float in one
// preset and as an integer in another.
func (s *PresetShapeTestSuite) TestAsFloat() {
	tests := []struct {
		name string
		in   any
		want float64
		ok   bool
	}{
		{name: "a double-width float", in: 1.5, want: 1.5, ok: true},
		{name: "a single-width float", in: float32(2.5), want: 2.5, ok: true},
		{name: "a one-byte integer", in: int8(3), want: 3, ok: true},
		{name: "an eight-byte integer", in: int64(4), want: 4, ok: true},
		{name: "an unsigned byte", in: uint8(5), want: 5, ok: true},
		{name: "eight unsigned bytes", in: uint64(6), want: 6, ok: true},
		{name: "something that is not a number", in: "text"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := asFloat(tt.in)

			s.Require().Equal(tt.ok, ok)

			if tt.ok {
				s.Require().InDelta(tt.want, got, 0.001)
			}
		})
	}
}

func TestPresetShapeTestSuite(t *testing.T) {
	suite.Run(t, new(PresetShapeTestSuite))
}
