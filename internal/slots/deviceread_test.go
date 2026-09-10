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

package slots

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// DeviceReadTestSuite reads what an HX Stomp actually answered.
//
// pkg/sdk/device/wire/testdata/preset.bin is one slot as the hardware handed it back.
// Everything from the wire to a rig runs here, so the live path is covered by
// a real answer rather than by a device being plugged in.
type DeviceReadTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *DeviceReadTestSuite) SetupSuite() {
	cat, err := catalogview.Open("")
	s.Require().NoError(err)

	s.cat = cat
}

func (s *DeviceReadTestSuite) capture() []byte { return s.answerFrom("preset.bin") }

// answerFrom returns one slot as the hardware sent it.
func (s *DeviceReadTestSuite) answerFrom(name string) []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "pkg", "sdk", "device", "wire", "testdata", name))
	s.Require().NoError(err)

	return raw
}

// encode renders a document the way a device would.
func (s *DeviceReadTestSuite) encode(doc map[int8]any) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeString("l6-helix\x00"))
	s.Require().NoError(enc.EncodeString("offsets"))
	s.Require().NoError(enc.Encode(doc))

	return buf.Bytes()
}

// block renders one chain entry naming the given model.
func block(model int) map[int8]any {
	return map[int8]any{
		19: 6,
		20: map[int8]any{24: map[int8]any{25: model}, 10: true},
	}
}

// empty is a slot holding no chain at all.
func (s *DeviceReadTestSuite) empty() []byte {
	return s.encode(map[int8]any{0: map[int8]any{22: []any{}}})
}

// bare is a chain and nothing else — no snapshots, no switches.
func (s *DeviceReadTestSuite) bare() []byte {
	return s.encode(map[int8]any{0: map[int8]any{22: []any{block(0)}}})
}

// unknownModel names a model no catalog reaches.
func (s *DeviceReadTestSuite) unknownModel() []byte {
	return s.encode(map[int8]any{0: map[int8]any{22: []any{block(99999)}}})
}

// TestWriteDeviceRig turns what a device answered into a rig.
func (s *DeviceReadTestSuite) TestWriteDeviceRig() {
	tests := []struct {
		name string
		// which answer to read: the capture unless a case says otherwise.
		answer string
		opts   DeviceOptions
		// a writer that fails, so a rig nobody can read is an error.
		deaf bool

		contains []string
		absent   []string
		err      bool
		errText  string
	}{
		{
			name: "a slot the device holds",
			opts: DeviceOptions{Slot: 79, Name: "BAS:SVT Nrm"},
			contains: []string{
				"schema: RigSpec",
				"name: BAS:SVT Nrm",

				// The slot is a factory preset, so what it holds is known
				// before it is decoded.
				"gear: Ampeg SVT® (normal channel)",
				"gear: 8x10 Ampeg SVT-E",
				"role: amp",

				// Read from the device, and nothing else in a rig records
				// them.
				"snapshots:",
				"SNAPSHOT 1",

				// What the device wraps the chain in, named the way a preset
				// names it. The models come from the catalog, because a
				// device knows which inputs and outputs are its own and does
				// not say.
				"dsp0.inputA",
				"'@model': HelixStomp_AppDSPFlowInput",
				"threshold: -48",
				"dsp0.split",
				"'@model': HD2_AppDSPFlowSplitY",
				"dsp0.join",

				// What the expression pedal moves, named rather than
				// numbered. The device stores parameter 0 of the block at
				// grid position 2, and only the catalog turns that into the
				// volume block's Pedal.
				"controllers:",
				"controller: 2",
				"parameter: Pedal",
				"block: 1",
			},
		},
		{
			// The device stores the two as one block. A preset stores the amp
			// with a `@cab` and the cabinet as a sibling, so both have to
			// come out.
			name:   "an amp carrying its own cabinet",
			answer: "switches.bin",
			opts:   DeviceOptions{Slot: 24},
			contains: []string{
				"dsp0.cab0",
				"dsp0.cab1",
				"'@model': HD2_Cab1x15TucknGo",
				"'@cab': cab0",
				"'@type': 3",
				"'@mic': 10",
			},
		},
		{
			name:     "a slot the device did not name",
			contains: []string{"slot 01A"},
		},
		{
			// A slot holding nothing is not a rig: it names no gear, and a
			// rig holds at least one thing.
			name:     "a slot holding nothing",
			answer:   "empty",
			opts:     DeviceOptions{Slot: 4},
			contains: []string{"is empty"},
		},
		{
			name:     "a chain with no snapshots and no switches",
			answer:   "bare",
			contains: []string{"schema: RigSpec"},
			absent:   []string{"snapshots:", "footswitches:"},
		},
		{
			name:    "an answer that is not a preset",
			answer:  "nonsense",
			opts:    DeviceOptions{Slot: 3},
			errText: "slot 02A",
		},
		{
			name: "a catalog it cannot open",
			opts: DeviceOptions{
				Slot: 0, CatalogPath: filepath.Join("testdata", "nope.json"),
			},
			err: true,
		},
		{
			name:   "a model the catalog cannot name",
			answer: "unknown",
			err:    true,
		},
		{
			// A catalog whose model table names an empty model produces a
			// chain with no gear in it, which is not a rig. Saying so beats
			// writing a document that claims to be one.
			name:   "a chain that is not a rig",
			answer: "bare",
			opts: DeviceOptions{
				Slot:        0,
				CatalogPath: filepath.Join("testdata", "unnamed.catalog.json"),
			},
			errText: "slot 01A",
		},
		{name: "a writer that fails", deaf: true, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var answer []byte

			switch tt.answer {
			case "":
				answer = s.capture()
			case "empty":
				answer = s.empty()
			case "bare":
				answer = s.bare()
			case "unknown":
				answer = s.unknownModel()
			case "nonsense":
				answer = []byte("nonsense")
			default:
				answer = s.answerFrom(tt.answer)
			}

			var out bytes.Buffer

			w := io.Writer(&out)
			if tt.deaf {
				w = &brokenWriter{}
			}

			err := writeDeviceRig(w, answer, tt.opts)

			if tt.err || tt.errText != "" {
				s.Require().Error(err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(out.String(), unwanted)
			}
		})
	}
}

func (s *DeviceReadTestSuite) TestARigReadOffTheDeviceRebuildsItsRouting() {
	// The claim this closes: a rig read over USB used to carry no routing, so
	// compiling it fell back to whatever preset it was built into. It now
	// carries the device's own, and building it puts that back.
	dir := s.T().TempDir()
	rigPath := filepath.Join(dir, "rig.yaml")
	out := filepath.Join(dir, "out.hlx")

	var buf bytes.Buffer
	s.Require().NoError(writeDeviceRig(&buf, s.capture(), DeviceOptions{Slot: 0}))
	s.Require().NoError(os.WriteFile(rigPath, buf.Bytes(), 0o600))

	s.Require().NoError(Compile(&bytes.Buffer{}, CompileOptions{
		RigPath: rigPath, OutputPath: out,
	}))

	built, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	got := string(built)
	s.Require().Contains(got, `"@model": "HelixStomp_AppDSPFlowInput"`)
	s.Require().Contains(got, `"threshold": -48`)
	s.Require().Contains(got, `"@model": "HD2_AppDSPFlowSplitY"`)
	s.Require().Contains(got, `"@model": "HD2_AppDSPFlowJoin"`)
}

func TestDeviceReadTestSuite(t *testing.T) {
	suite.Run(t, new(DeviceReadTestSuite))
}
