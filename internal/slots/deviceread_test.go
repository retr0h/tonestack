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
)

// DeviceReadTestSuite reads what an HX Stomp actually answered.
//
// pkg/sdk/wire/testdata/preset.bin is one slot as the hardware handed it back.
// Everything from the wire to a rig runs here, so the live path is covered by
// a real answer rather than by a device being plugged in.
type DeviceReadTestSuite struct {
	suite.Suite
}

func (s *DeviceReadTestSuite) capture() []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "pkg", "sdk", "wire", "testdata", "preset.bin"))
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

	// The untouched preset it was assembled into is not this slot, so none of
	// its state may be presented as though it were.
	s.Require().NotContains(got, "\ndevice:")
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

func TestDeviceReadTestSuite(t *testing.T) {
	suite.Run(t, new(DeviceReadTestSuite))
}
