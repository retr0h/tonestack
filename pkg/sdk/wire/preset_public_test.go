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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// PresetPublicTestSuite reads what an HX Stomp actually answered.
//
// testdata/preset.bin is one slot as the hardware handed it back, captured
// over USB. Nothing else in this repository describes the wire format, so a
// real answer is the only thing that can say whether the reading is right.
type PresetPublicTestSuite struct {
	suite.Suite
}

func (s *PresetPublicTestSuite) capture() []byte {
	return s.captureNamed("preset.bin")
}

// captureNamed returns one slot as an HX Stomp actually sent it.
func (s *PresetPublicTestSuite) captureNamed(name string) []byte {
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	s.Require().NoError(err)

	return raw
}

// withControllers rebuilds the captured preset with a different controller
// section, so shapes a device never sends can still be read.
func (s *PresetPublicTestSuite) withControllers(section any) []byte {
	doc, err := wire.DecodeDocument(s.capture())
	s.Require().NoError(err)

	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	enc.SetCustomStructTag("msgpack")
	s.Require().NoError(enc.Encode(section))

	doc.SetSection(4, buf.Bytes())

	return doc.Encode()
}

// TestDecodePreset reads a preset a device sent.
func (s *PresetPublicTestSuite) TestDecodePreset() {
	tests := []struct {
		name string
		// one of this suite's captures, or bytes written by hand.
		file string
		body []byte
		// the captured preset rebuilt with another controller section.
		custom  bool
		section any

		wantBlocks int
		// the model each block names, by its place in the reading.
		models map[int]int
		// where the device put a block, which is not where it falls in the
		// chain.
		index map[int]int
		// how many parameters a block carries.
		values map[int]int
		// blocks that must read as switched on, and as switched off.
		on  []int
		off []int
		// blocks carrying a cabinet of their own.
		paired      int
		controllers []wire.DeviceController
		err         bool
	}{
		{
			// The slot is a factory preset called "BAS:SVT Nrm", so the
			// models it names are known before decoding: a volume pedal, an
			// LA-2A, two pitch blocks, the SVT's normal channel and the 8x10
			// it is voiced with.
			name:       "a factory preset",
			file:       "preset.bin",
			wantBlocks: 6,
			models: map[int]int{
				0: 261,
				// The SVT normal channel, and the 8x10 it is paired with.
				4: 5,
				5: 74,
			},
			// A preset leaves gaps in its layout, and footswitches address
			// blocks by this.
			index: map[int]int{0: 2},
			// An amp has twelve, and their names come from the catalog's
			// model table rather than from anything the device sent.
			values: map[int]int{4: 12},
			on:     []int{0},
			// A bypassed block is still a block.
			off: []int{2},
			// The captured preset carries exactly one assignment, and it is
			// what established the layout: slot 27B read off an HX Stomp
			// beside the same slot exported from HX Edit, which names it as
			// block0's Pedal with min 0, max 1, controller 2.
			controllers: []wire.DeviceController{
				{Controller: 2, Block: 2, Param: 0, Min: 0, Max: 1},
			},
		},
		{
			// A device stores an amplifier and its cabinet as one block.
			name:   "a slot holding two amplifiers",
			file:   "switches.bin",
			paired: 2,
		},
		{name: "a slot holding nothing", file: "empty.bin"},
		{name: "nothing at all", err: true},
		{name: "something else entirely", body: []byte{0xc0}, err: true},
		{
			name: "the header and no more",
			body: []byte("\xa9l6-helix\x00"),
			err:  true,
		},
		{
			name: "a header and offsets but no document",
			body: []byte("\xa9l6-helix\x00\xa1x"),
			err:  true,
		},
		{
			name: "a document that is not a map",
			body: []byte("\xa9l6-helix\x00\xa1x\xc3"),
			err:  true,
		},
		// Every shape the controller decoder refuses, since a preset carrying
		// one assignment exercises none of them.
		{
			name:       "a controller section that is not an array",
			custom:     true,
			section:    42,
			wantBlocks: 6,
		},
		{
			name:       "a controller holding something that is not a list",
			custom:     true,
			section:    []any{42},
			wantBlocks: 6,
		},
		{
			name:       "an assignment that is not a map",
			custom:     true,
			section:    []any{[]any{"nonsense"}},
			wantBlocks: 6,
		},
		{
			name:       "an assignment naming no parameter",
			custom:     true,
			section:    []any{[]any{map[any]any{int8(1): map[any]any{}}}},
			wantBlocks: 6,
		},
		{
			name:       "one with no body",
			custom:     true,
			section:    []any{[]any{map[any]any{int8(0): int8(0)}}},
			wantBlocks: 6,
		},
		{
			name:   "one naming no block",
			custom: true,
			section: []any{[]any{map[any]any{
				int8(0): int8(0),
				int8(1): map[any]any{int8(2): 0.0},
			}}},
			wantBlocks: 6,
		},
		{
			name:   "travel a device wrote as whole numbers",
			custom: true,
			section: []any{nil, nil, []any{map[any]any{
				int8(0): int8(1),
				int8(1): map[any]any{
					int8(0): int8(5),
					int8(2): int8(0),
					int8(3): int8(1),
					int8(6): map[any]any{int8(41): true},
				},
			}}},
			wantBlocks: 6,
			controllers: []wire.DeviceController{{
				Controller: 2, Block: 5, Param: 1, Min: 0, Max: 1,
				NoSnapshot: true,
			}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			body := tt.body

			switch {
			case tt.custom:
				body = s.withControllers(tt.section)
			case tt.file != "":
				body = s.captureNamed(tt.file)
			}

			got, err := wire.DecodePreset(body)

			if tt.err {
				s.Require().ErrorIs(err, wire.ErrNotAPreset)

				return
			}

			s.Require().NoError(err)

			if tt.wantBlocks > 0 {
				s.Require().Len(got.Blocks, tt.wantBlocks)
			}

			for at, want := range tt.models {
				s.Require().Equal(want, got.Blocks[at].Model)
			}

			for at, want := range tt.index {
				s.Require().Equal(want, got.Blocks[at].Index)
			}

			for at, want := range tt.values {
				s.Require().Len(got.Blocks[at].Values, want)
			}

			for _, at := range tt.on {
				s.Require().True(got.Blocks[at].Enabled)
			}

			for _, at := range tt.off {
				s.Require().False(got.Blocks[at].Enabled)
			}

			var paired int

			for _, b := range got.Blocks {
				if len(b.Cab) == 0 {
					continue
				}

				paired++

				// Five settings the cabinet model has names for, and a
				// microphone after them.
				s.Require().Equal(5, b.CabNamed)
				s.Require().Len(b.Cab, 6)
			}

			s.Require().Equal(tt.paired, paired)
			s.Require().Equal(tt.controllers, got.Controllers)
		})
	}
}

// TestDecodeLoaded covers what a device answers when asked what it is
// playing, which is the only honest signal that a select has finished.
func (s *PresetPublicTestSuite) TestDecodeLoaded() {
	tests := []struct {
		name   string
		result any
		want   wire.Loaded
		err    bool
	}{
		{
			name: "a device naming what it plays",
			result: map[any]any{
				int8(107): uint8(0),
				int8(108): uint8(99),
				int8(109): "Chunky Monkey",
			},
			want: wire.Loaded{Setlist: 0, Slot: 99, Name: "Chunky Monkey"},
		},
		{
			name: "one that names no preset",
			result: map[any]any{
				int8(107): uint8(0),
				int8(108): uint8(7),
			},
			want: wire.Loaded{Setlist: 0, Slot: 7},
		},
		{
			name:   "an answer that is not a map",
			result: "nonsense",
			err:    true,
		},
		{
			name:   "one naming no setlist",
			result: map[any]any{int8(108): uint8(7)},
			err:    true,
		},
		{
			name:   "one naming no slot",
			result: map[any]any{int8(107): uint8(0)},
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := wire.DecodeLoaded(tt.result)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

func TestPresetPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PresetPublicTestSuite))
}
