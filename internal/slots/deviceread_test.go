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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/catalog"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// DeviceReadTestSuite reads what an HX Stomp actually answered.
//
// pkg/sdk/wire/testdata/preset.bin is one slot as the hardware handed it back.
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
		filepath.Join("..", "..", "pkg", "sdk", "wire", "testdata", name))
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

func (s *DeviceReadTestSuite) TestADeviceAnswerBecomesARig() {
	var out bytes.Buffer

	s.Require().NoError(writeDeviceRig(&out, s.capture(),
		DeviceOptions{Slot: 79, Name: "BAS:SVT Nrm"}))

	got := out.String()
	s.Require().Contains(got, "schema: RigSpec")
	s.Require().Contains(got, "name: BAS:SVT Nrm")

	// The slot is a factory preset, so what it holds is known before it is
	// decoded.
	s.Require().Contains(got, "gear: Ampeg SVT® (normal channel)")
	s.Require().Contains(got, "gear: 8x10 Ampeg SVT-E")
	s.Require().Contains(got, "role: amp")

	// Read from the device, and nothing else in a rig records them.
	s.Require().Contains(got, "snapshots:")
	s.Require().Contains(got, "SNAPSHOT 1")

	// What the device wraps the chain in, named the way a preset names it.
	// The models come from the catalog, because a device knows which inputs
	// and outputs are its own and does not say.
	s.Require().Contains(got, "dsp0.inputA")
	s.Require().Contains(got, "'@model': HelixStomp_AppDSPFlowInput")
	s.Require().Contains(got, "threshold: -48")
	s.Require().Contains(got, "dsp0.split")
	s.Require().Contains(got, "'@model': HD2_AppDSPFlowSplitY")
	s.Require().Contains(got, "dsp0.join")

	// What the expression pedal moves, named rather than numbered. The device
	// stores parameter 0 of the block at grid position 2, and only the
	// catalog turns that into the volume block's Pedal.
	s.Require().Contains(got, "controllers:")
	s.Require().Contains(got, "controller: 2")
	s.Require().Contains(got, "parameter: Pedal")
	s.Require().Contains(got, "block: 1")
}

func (s *DeviceReadTestSuite) TestAnAmpCarryingItsOwnCabinet() {
	// The device stores the two as one block. A preset stores the amp with a
	// `@cab` and the cabinet as a sibling, so both have to come out.
	var out bytes.Buffer

	s.Require().NoError(writeDeviceRig(&out, s.answerFrom("switches.bin"),
		DeviceOptions{Slot: 24}))

	got := out.String()
	s.Require().Contains(got, "dsp0.cab0")
	s.Require().Contains(got, "dsp0.cab1", "two amps, two cabinets")
	s.Require().Contains(got, "'@model': HD2_Cab1x15TucknGo")
	s.Require().Contains(got, "'@cab': cab0")
	s.Require().Contains(got, "'@type': 3", "an amp carrying a cabinet")
	s.Require().Contains(got, "'@mic': 10")
}

func (s *DeviceReadTestSuite) TestASlotWithNoNameIsCalledBySlot() {
	var out bytes.Buffer

	s.Require().NoError(writeDeviceRig(&out, s.capture(), DeviceOptions{Slot: 0}))
	s.Require().Contains(out.String(), "slot 01A")
}

func (s *DeviceReadTestSuite) TestReportsAnAnswerThatIsNotAPreset() {
	err := writeDeviceRig(&bytes.Buffer{}, []byte("nonsense"), DeviceOptions{Slot: 3})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "slot 02A")
}

func (s *DeviceReadTestSuite) TestReportsACatalogItCannotOpen() {
	err := writeDeviceRig(&bytes.Buffer{}, s.capture(),
		DeviceOptions{Slot: 0, CatalogPath: filepath.Join("testdata", "nope.json")})

	s.Require().Error(err)
}

func (s *DeviceReadTestSuite) TestReportsAWriterThatFails() {
	err := writeDeviceRig(&brokenWriter{}, s.capture(), DeviceOptions{Slot: 0})

	s.Require().Error(err)
}

func (s *DeviceReadTestSuite) TestAnEmptySlotSaysSo() {
	// A slot holding nothing is not a rig: it names no gear, and a rig holds
	// at least one thing.
	var out bytes.Buffer

	s.Require().NoError(writeDeviceRig(&out, s.empty(), DeviceOptions{Slot: 4}))
	s.Require().Contains(out.String(), "is empty")
}

func (s *DeviceReadTestSuite) TestASlotWithNoSnapshotsOrSwitches() {
	var out bytes.Buffer

	s.Require().NoError(writeDeviceRig(&out, s.bare(), DeviceOptions{Slot: 0}))

	got := out.String()
	s.Require().NotContains(got, "snapshots:")
	s.Require().NotContains(got, "footswitches:")
	s.Require().Contains(got, "schema: RigSpec")
}

func (s *DeviceReadTestSuite) TestReportsAModelTheCatalogCannotName() {
	s.Require().Error(writeDeviceRig(&bytes.Buffer{}, s.unknownModel(),
		DeviceOptions{Slot: 0}))
}

func (s *DeviceReadTestSuite) TestReportsAChainThatIsNotARig() {
	// A catalog whose model table names an empty model produces a chain with
	// no gear in it, which is not a rig. Saying so beats writing a document
	// that claims to be one.
	err := writeDeviceRig(&bytes.Buffer{}, s.bare(), DeviceOptions{
		Slot:        0,
		CatalogPath: filepath.Join("testdata", "unnamed.catalog.json"),
	})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "slot 01A")
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

// TestControllersOf covers naming what a controller moves, and the cases
// where it cannot be named.
func (s *DeviceReadTestSuite) TestControllersOf() {
	// The block at grid position 2 in the captured preset is a volume pedal,
	// whose parameters are Pedal and VolumeTaper in that order.
	block := wire.DeviceBlock{Index: 2, Model: 261}

	tests := []struct {
		name  string
		got   wire.DevicePreset
		want  *[]riggen.Controller
		named string
	}{
		{
			name: "an expression pedal on a parameter",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 0, Min: 0, Max: 1},
				},
			},
			named: "Pedal",
		},
		{
			name: "one snapshots leave alone",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 1, NoSnapshot: true},
				},
			},
			named: "VolumeTaper",
		},
		{
			name: "a preset assigning none",
			got:  wire.DevicePreset{Blocks: []wire.DeviceBlock{block}},
		},
		{
			name: "one naming a block the chain does not hold",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 9, Param: 0},
				},
			},
		},
		{
			name: "one naming a parameter past what the model has",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 40},
				},
			},
		},
		{
			name: "one on a model this catalog cannot name",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{{Index: 2, Model: 999999}},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 0},
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := controllersOf(tt.got, s.cat)

			if tt.named == "" {
				s.Require().Nil(got, "nothing that cannot be named is written")

				return
			}

			s.Require().NotNil(got)
			s.Require().Len(*got, 1)
			s.Require().Equal(tt.named, (*got)[0].Parameter)

			// A grid position is not a place along the path, and a rig counts
			// the way the chain does.
			s.Require().Equal(1, (*got)[0].Block)
		})
	}
}
