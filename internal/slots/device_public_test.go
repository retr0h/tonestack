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

package slots_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// DevicePublicTestSuite covers reading a device, with no device attached.
//
// The answers are real: pkg/sdk/wire/testdata holds three slots exactly as an
// HX Stomp handed them back — one full preset, one whose switches somebody
// labelled and coloured, and one empty slot.
type DevicePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	dev  *mocks.MockEditor
}

func (s *DevicePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.dev = mocks.NewMockEditor(s.ctrl)
	s.dev.EXPECT().Model().Return(sdk.Model{Name: "HX Stomp"}).AnyTimes()
}

func (s *DevicePublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// answer returns one slot as the hardware sent it.
func (s *DevicePublicTestSuite) answer(name string) string {
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "pkg", "sdk", "wire", "testdata", name))
	s.Require().NoError(err)

	return string(raw)
}

// listing is what the device says a setlist holds.
func (s *DevicePublicTestSuite) listing() []wire.Preset {
	return []wire.Preset{
		{Slot: 0, Name: "Chunky Monkey"},
		{Slot: 1, Name: "New Preset"},
		{Slot: 24, Name: "B15 Eras"},
	}
}

func (s *DevicePublicTestSuite) TestListsWhatADeviceHolds() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)

	var out bytes.Buffer
	s.Require().NoError(
		slots.ListWith(context.Background(), &out, s.dev, slots.DeviceOptions{}))

	got := out.String()
	s.Require().Contains(got, "01A")
	s.Require().Contains(got, "Chunky Monkey")
	s.Require().Contains(got, "09A")

	// A device names an untouched slot rather than leaving it blank, so what
	// counts as empty is the name it was shipped with.
	s.Require().NotContains(got, "New Preset")
}

func (s *DevicePublicTestSuite) TestListsEmptySlotsWhenAsked() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)

	var out bytes.Buffer
	s.Require().NoError(slots.ListWith(
		context.Background(), &out, s.dev, slots.DeviceOptions{All: true}))

	s.Require().Contains(out.String(), "New Preset")
}

func (s *DevicePublicTestSuite) TestReportsAListingItCannotGet() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(nil, errors.New("boom"))

	err := slots.ListWith(
		context.Background(), &bytes.Buffer{}, s.dev, slots.DeviceOptions{})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "listing presets")
}

func (s *DevicePublicTestSuite) TestReadsOneSlotAsARig() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 24).Return(s.answer("switches.bin"), nil)

	var out bytes.Buffer
	s.Require().NoError(slots.ShowWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 24}))

	got := out.String()
	s.Require().Contains(got, "schema: RigSpec")
	s.Require().Contains(got, "name: B15 Eras", "the name comes from the listing")

	// Somebody labelled and coloured these switches, and a rig carries what
	// the pedal shows rather than what the block is called.
	s.Require().Contains(got, "label: Drive")
	s.Require().Contains(got, "gear: Teemah!")
	s.Require().Contains(got, "led: light orange")
	s.Require().Contains(got, "switch: 2")
}

func (s *DevicePublicTestSuite) TestASlotHoldingNothing() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 1).Return(s.answer("empty.bin"), nil)

	var out bytes.Buffer
	s.Require().NoError(slots.ShowWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 1}))

	s.Require().Contains(out.String(), "is empty")
}

func (s *DevicePublicTestSuite) TestASlotTheListingDoesNotReach() {
	// The name is a convenience. A slot beyond what the listing returned is
	// still read, and named by where it sits.
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 99).Return(s.answer("preset.bin"), nil)

	var out bytes.Buffer
	s.Require().NoError(slots.ShowWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 99}))

	s.Require().Contains(out.String(), "slot 34A")
}

func (s *DevicePublicTestSuite) TestCarriesOnWithoutAListing() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(nil, errors.New("boom"))
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer("preset.bin"), nil)

	var out bytes.Buffer
	s.Require().NoError(slots.ShowWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 0}))

	s.Require().Contains(out.String(), "slot 01A")
}

func (s *DevicePublicTestSuite) TestReportsASlotItCannotRead() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 3).Return(nil, errors.New("boom"))

	err := slots.ShowWith(context.Background(), &bytes.Buffer{}, s.dev,
		slots.DeviceOptions{Slot: 3})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "slot 02A")
}

func (s *DevicePublicTestSuite) TestAnAnswerThatIsNotAPreset() {
	// Nothing guarantees what comes off a wire. Saying what arrived beats
	// printing a rig that would be wrong.
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(map[any]any{1: 2}, nil)

	var out bytes.Buffer
	s.Require().NoError(slots.ShowWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 0}))

	s.Require().Contains(out.String(), "map with 1 keys")
}

func (s *DevicePublicTestSuite) TestKeepsTheAnswerWhenAsked() {
	path := filepath.Join(s.T().TempDir(), "slot.bin")
	s.T().Setenv("TONESTACK_USB_DUMP", path)

	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer("preset.bin"), nil)

	s.Require().NoError(slots.ShowWith(context.Background(), &bytes.Buffer{},
		s.dev, slots.DeviceOptions{Slot: 0}))

	body, err := os.ReadFile(path) //nolint:gosec // a path this test wrote
	s.Require().NoError(err)
	s.Require().Equal(s.answer("preset.bin"), string(body), "kept verbatim")
}

func (s *DevicePublicTestSuite) TestReportsAnAnswerItCannotKeep() {
	s.T().Setenv("TONESTACK_USB_DUMP",
		filepath.Join(s.T().TempDir(), "no", "such", "dir.bin"))

	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer("preset.bin"), nil)

	s.Require().Error(slots.ShowWith(context.Background(), &bytes.Buffer{},
		s.dev, slots.DeviceOptions{Slot: 0}))
}

func (s *DevicePublicTestSuite) TestWritesOneSlotToAFile() {
	out := filepath.Join(s.T().TempDir(), "rig.yaml")

	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 24).Return(s.answer("switches.bin"), nil)

	var log bytes.Buffer
	s.Require().NoError(slots.ExportWith(context.Background(), &log, s.dev,
		slots.ExportOptions{Slot: 24, OutputPath: out}))

	s.Require().Contains(log.String(), "wrote "+out)

	body, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)
	s.Require().Contains(string(body), "schema: RigSpec")
}

// TestReportsASlotHoldingNothing covers the answer a device gives for an
// empty slot: no document at all.
//
// It used to reach a diagnostic left over from before the format was decoded,
// which printed what shape had arrived. An export then wrote that prose into
// the file and reported success.
func (s *DevicePublicTestSuite) TestReportsASlotHoldingNothing() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil).Times(2)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 4).Return(nil, nil).Times(2)

	var out bytes.Buffer

	err := slots.ShowWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 4})
	s.Require().ErrorIs(err, slots.ErrEmptySlot)

	// And an export writes nothing rather than a file that is not a preset.
	path := filepath.Join(s.T().TempDir(), "empty.hlx")

	s.Require().ErrorIs(slots.ExportWith(context.Background(), &out, s.dev,
		slots.ExportOptions{Slot: 4, As: "hlx", OutputPath: path}),
		slots.ErrEmptySlot)

	s.Require().NoFileExists(path, "an empty slot leaves no file behind")
}

// TestWritesOneSlotAsTheDevicesOwnFile covers `--as hlx`, which the device
// path ignored: it wrote a rig whatever was asked for.
//
// The positions are the check that matters. A preset counts its blocks along
// the path and a device counts across a grid holding its routing too, so an
// export carrying the device's numbers is not the file HX Edit writes. Slot
// 27B exported from HX Edit holds these six at 1 to 6, and preset.bin is that
// same slot as the device sent it.
func (s *DevicePublicTestSuite) TestWritesOneSlotAsTheDevicesOwnFile() {
	out := filepath.Join(s.T().TempDir(), "slot.hlx")

	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer("preset.bin"), nil)

	s.Require().NoError(slots.ExportWith(context.Background(), &bytes.Buffer{},
		s.dev, slots.ExportOptions{Slot: 0, As: "hlx", OutputPath: out}))

	body, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	// A tone holds more than blocks — the globals, the snapshots, the routing
	// — so entries are read one at a time rather than through one shape.
	var doc struct {
		Data struct {
			Tone map[string]map[string]json.RawMessage `json:"tone"`
		} `json:"data"`
	}

	s.Require().NoError(json.Unmarshal(body, &doc))

	at := map[int]string{}

	for key, raw := range doc.Data.Tone["dsp0"] {
		if !strings.HasPrefix(key, "block") {
			continue
		}

		var entry struct {
			Model    string `json:"@model"`
			Position *int   `json:"@position"`
		}

		s.Require().NoError(json.Unmarshal(raw, &entry))
		s.Require().NotNil(entry.Position, "%s states no position", key)

		at[*entry.Position] = entry.Model
	}

	s.Require().Equal(map[int]string{
		1: "HD2_VolPanVol",
		2: "HD2_CompressorLAStudioComp",
		3: "HD2_FM4Growler",
		4: "HD2_DM4BassOctaver",
		5: "HD2_AmpSVBeastNrm",
		6: "HD2_Cab8x10SVBeast",
	}, at, "what HX Edit writes for this slot")

	s.Require().Contains(doc.Data.Tone["dsp0"], "inputA",
		"the routing a device wraps a chain in comes too")
}

func (s *DevicePublicTestSuite) TestReportsAFileItCannotWrite() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer("preset.bin"), nil)

	err := slots.ExportWith(context.Background(), &bytes.Buffer{}, s.dev,
		slots.ExportOptions{
			Slot:       0,
			OutputPath: filepath.Join(s.T().TempDir(), "no", "such", "dir.yaml"),
		})

	s.Require().Error(err)
}

func (s *DevicePublicTestSuite) TestReportsASlotItCannotExport() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(nil, errors.New("boom"))

	s.Require().Error(slots.ExportWith(context.Background(), &bytes.Buffer{},
		s.dev, slots.ExportOptions{Slot: 0, OutputPath: "x.yaml"}))
}

// stand puts a session in place of the one that needs hardware, and takes it
// away again.
func (s *DevicePublicTestSuite) stand(dev sdk.Editor, err error) func() {
	restore := *slots.OpenDevice
	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return dev, err
	}

	return func() { *slots.OpenDevice = restore }
}

func (s *DevicePublicTestSuite) TestTheCommandsThatFindTheirOwnDevice() {
	// The three entry points are one line each — find a session, hand it on,
	// release it — and the only line in the package that needs hardware.
	out := filepath.Join(s.T().TempDir(), "rig.yaml")

	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil).Times(3)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).
		Return(s.answer("preset.bin"), nil).Times(2)
	s.dev.EXPECT().Close().Times(3)

	defer s.stand(s.dev, nil)()

	ctx := context.Background()

	s.Require().NoError(slots.ListDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}))
	s.Require().NoError(slots.ShowDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}))
	s.Require().NoError(slots.ExportDevice(ctx, &bytes.Buffer{},
		slots.ExportOptions{OutputPath: out}))
}

func (s *DevicePublicTestSuite) TestReportsADeviceItCannotOpen() {
	defer s.stand(nil, errors.New("no device found"))()

	ctx := context.Background()

	for _, err := range []error{
		slots.ListDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}),
		slots.ShowDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}),
		slots.ExportDevice(ctx, &bytes.Buffer{}, slots.ExportOptions{}),
	} {
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "no device found")
	}
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
