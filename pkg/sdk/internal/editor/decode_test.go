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

package editor

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// DecodeTestSuite covers turning a device's answer into a chain.
//
// The device names nothing: a block is a number into its own model table and
// its parameters are a bare array. Everything here is about putting the names
// back, and about saying so plainly when they cannot be.
type DecodeTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *DecodeTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// TestChainOf turns a device's answer into a chain.
func (s *DecodeTestSuite) TestChainOf() {
	tests := []struct {
		name string
		// what the slot is called, and the blocks the device sent.
		title  string
		blocks []wire.DeviceBlock
		// a catalog with no model table at all.
		bare bool

		wantName string
		// where each block sits along the path, and which model it names.
		wantPos    map[int]int
		wantModels map[int]catalog.ModelID
		// the parameters a block must carry, must not, and how many in all.
		params  map[int]map[string]catalog.ParamValue
		absent  map[int][]string
		howMany map[int]int
		// blocks that must read as switched on, and as switched off.
		on      []int
		off     []int
		errText string
	}{
		{
			// A device lays blocks out on a grid holding its routing as well,
			// so a chain of two can sit at 5 and 13 with the gaps kept. A
			// preset counts along the path instead, and the two differ by
			// one: slot 27B exported from HX Edit holds its six blocks at 1
			// to 6 where the device sent the same six at 2 to 7.
			name: "blocks counted the way a preset counts them",
			blocks: []wire.DeviceBlock{
				{Index: 5, Model: 5},
				{Index: 13, Model: 74},
			},
			wantPos: map[int]int{0: 4, 1: 12},
		},
		{
			name:  "what the device numbered",
			title: "BAS:SVT Nrm",
			blocks: []wire.DeviceBlock{
				{Model: 5, Values: []any{0.27, 0.65}, Enabled: true},
				{Model: 74, Values: []any{1.0}, Enabled: false},
			},
			wantName: "BAS:SVT Nrm",
			wantModels: map[int]catalog.ModelID{
				0: "HD2_AmpSVBeastNrm",
				1: "HD2_Cab8x10SVBeast",
			},
			params: map[int]map[string]catalog.ParamValue{
				0: {"Drive": catalog.Float(0.27), "Bass": catalog.Float(0.65)},
			},
			on: []int{0},
			// A bypassed block is still a block.
			off: []int{1},
		},
		{
			// A device names a mono and a stereo instance separately; the
			// catalog names the model once, the way Line 6's own model files
			// do.
			name:       "a mono instance of a model named once",
			blocks:     []wire.DeviceBlock{{Model: 120, Enabled: true}},
			wantModels: map[int]catalog.ModelID{0: "HD2_CompressorLAStudioComp"},
		},
		{
			// Symbols cover every Helix; a Stomp has no second effects loop.
			// Keeping the device's own name means the rig still rebuilds it
			// exactly.
			name:       "a model this device does not have",
			blocks:     []wire.DeviceBlock{{Model: -1}},
			wantModels: map[int]catalog.ModelID{0: "HD2_FXLoopMono3"},
		},
		{
			// A device mixes numbers, switches and enumerated positions in
			// one array, and narrowing them all to numbers turns every switch
			// off. The wire layer has already reduced them to those three
			// kinds.
			name: "a value of every kind, in the shape it arrived in",
			blocks: []wire.DeviceBlock{
				{Model: 5, Values: []any{0.27, true, int64(2), "unreadable"}},
			},
			params: map[int]map[string]catalog.ParamValue{0: {
				"Drive": catalog.Float(0.27),
				"Bass":  catalog.Bool(true),
				"Mid":   catalog.Int(2),
				// An amp, and one carrying no cabinet of its own.
				"@type": catalog.Int(1),
			}},
			// A value of no known kind is left out.
			absent: map[int][]string{0: {"MidFreq", "@cab"}},
		},
		{
			// A device says less than the table names when it has nothing to
			// say. The one value it sent, plus the kind of block a preset
			// would call this.
			name:    "a truncated run of values",
			blocks:  []wire.DeviceBlock{{Model: 5, Values: []any{0.27}}},
			params:  map[int]map[string]catalog.ParamValue{0: {"Drive": catalog.Float(0.27)}},
			howMany: map[int]int{0: 2},
		},
		{
			name:    "a model the table does not reach",
			blocks:  []wire.DeviceBlock{{Model: 99999}},
			errText: "different release",
		},
		{
			name:    "a catalog with no model table",
			blocks:  []wire.DeviceBlock{{Model: 5}},
			bare:    true,
			errText: "catalog generate",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cat := s.cat
			if tt.bare {
				cat = &catalog.Catalog{}
			}

			blocks := tt.blocks
			for i, b := range blocks {
				// A model this device does not have, named rather than
				// numbered because where it sits is a fact about the
				// catalog.
				if b.Model == -1 {
					blocks[i].Model = indexOf(s.cat, "HD2_FXLoopMono3")
				}
			}

			title := tt.title
			if title == "" {
				title = "x"
			}

			got, err := Chain(title, wire.DevicePreset{Blocks: blocks}, cat)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got.Blocks, len(tt.blocks))

			if tt.wantName != "" {
				s.Require().Equal(tt.wantName, got.Name)
			}

			for at, want := range tt.wantPos {
				s.Require().Equal(want, got.Blocks[at].Pos)
			}

			for at, want := range tt.wantModels {
				s.Require().Equal(want, got.Blocks[at].Model)
			}

			for at, want := range tt.params {
				for key, value := range want {
					s.Require().Equal(value, got.Blocks[at].Params[key], key)
				}
			}

			for at, keys := range tt.absent {
				for _, key := range keys {
					s.Require().NotContains(got.Blocks[at].Params, key)
				}
			}

			for at, want := range tt.howMany {
				s.Require().Len(got.Blocks[at].Params, want)
			}

			for _, at := range tt.on {
				s.Require().True(got.Blocks[at].Enabled)
			}

			for _, at := range tt.off {
				s.Require().False(got.Blocks[at].Enabled)
			}
		})
	}
}

// indexOf finds where a model sits in the device's own table.
func indexOf(cat *catalog.Catalog, id catalog.ModelID) int {
	for i, sym := range cat.Symbols {
		if sym.ID == id {
			return i
		}
	}

	return -1
}

// TestMicAttr covers the value a cabinet sends past its named ones.
func (s *DecodeTestSuite) TestMicAttr() {
	sym := catalog.Symbol{Params: []string{"Level", "LowCut"}}

	tests := []struct {
		name   string
		values []any
		want   map[string]json.RawMessage
	}{
		{
			name:   "a cabinet sending one past its names",
			values: []any{0.5, 20.0, int64(11)},
			want:   map[string]json.RawMessage{"@mic": json.RawMessage("11")},
		},
		{
			name:   "a block sending exactly what it names",
			values: []any{0.5, 20.0},
		},
		{
			name:   "one sending fewer than it names",
			values: []any{0.5},
		},
		{
			// A float32 bit pattern can hold one, and encoding/json refuses
			// to write it. Dropping the attribute is better than refusing to
			// read the preset it came in.
			name:   "a value JSON cannot write",
			values: []any{0.5, 20.0, math.NaN()},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, micAttr(sym, tt.values))
		})
	}
}

func TestDecodeTestSuite(t *testing.T) {
	suite.Run(t, new(DecodeTestSuite))
}
