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
		{Slot: 79, Name: "BAS:SVT Nrm"},
	}
}

// full is the reads a listing makes: one per named slot, because a name says
// nothing about whether anything is in it.
func (s *DevicePublicTestSuite) full() {
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer("preset.bin"), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 24).Return(s.answer("switches.bin"), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 79).Return(s.answer("empty.bin"), nil)
}

// TestListWith writes out what a setlist holds.
func (s *DevicePublicTestSuite) TestListWith() {
	tests := []struct {
		name     string
		opts     slots.DeviceOptions
		expect   func()
		contains []string
		absent   []string
		says     string
	}{
		{
			name:     "every slot holding a chain, and what is in it",
			expect:   s.full,
			contains: []string{"01A", "Chunky Monkey", "09A", "amp", "2 in use"},
			// A slot nobody has named holds nothing, and one that is named
			// can still hold nothing. Neither belongs in a listing of what
			// somebody made.
			absent: []string{"New Preset", "BAS:SVT Nrm"},
		},
		{
			// Two kinds of empty, and the difference is worth seeing.
			name:     "the empty ones as well, when asked",
			opts:     slots.DeviceOptions{All: true},
			expect:   s.full,
			contains: []string{"New Preset", "BAS:SVT Nrm", "empty"},
		},
		{
			name: "a catalog that will not open",
			opts: slots.DeviceOptions{CatalogPath: "nowhere.json"},
			says: "nowhere.json",
		},
		{
			// A catalog generated from another release should not hide every
			// slot behind the first model it cannot name.
			name: "one whose model table this device has outgrown",
			opts: slots.DeviceOptions{
				All:         true,
				CatalogPath: filepath.Join("testdata", "unnamed.catalog.json"),
			},
			expect:   s.full,
			contains: []string{"0 in use"},
		},
		{
			name: "a slot the device will not read",
			expect: func() {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(nil, errors.New("boom"))
			},
			says: "boom",
		},
		{
			// One unreadable preset should not hide the hundred that read.
			name: "a preset that will not decode",
			expect: func() {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return("not a preset", nil)
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 24).Return(42, nil)
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 79).
					Return(s.answer("empty.bin"), nil)
			},
			opts:     slots.DeviceOptions{All: true},
			contains: []string{"0 in use"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)

			if tt.expect != nil {
				tt.expect()
			}

			var out bytes.Buffer

			err := slots.ListWith(context.Background(), &out, s.dev, tt.opts)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(out.String(), gone)
			}
		})
	}
}

// TestListWithReportsAListingItCannotGet covers the device refusing the one
// call the listing cannot do without.
func (s *DevicePublicTestSuite) TestListWithReportsAListingItCannotGet() {
	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(nil, errors.New("boom"))

	s.Require().ErrorContains(slots.ListWith(context.Background(),
		&bytes.Buffer{}, s.dev, slots.DeviceOptions{}), "listing presets")
}

// TestShowWith writes out one slot.
func (s *DevicePublicTestSuite) TestShowWith() {
	tests := []struct {
		name     string
		slot     int
		answer   any
		listing  error
		contains []string
		is       error
		says     string
	}{
		{
			name:   "a preset, as a rig",
			slot:   24,
			answer: s.answer("switches.bin"),
			contains: []string{
				"schema: RigSpec",
				// The name comes from the listing.
				"name: B15 Eras",
				// Somebody labelled and coloured these switches, and a rig
				// carries what the pedal shows rather than what the block is
				// called.
				"label: Drive", "gear: Teemah!", "led: light orange", "switch: 2",
			},
		},
		{
			name:     "one holding no blocks",
			slot:     1,
			answer:   s.answer("empty.bin"),
			contains: []string{"is empty"},
		},
		{
			// The name is a convenience. A slot beyond what the listing
			// returned is still read, and named by where it sits.
			name:     "one the listing does not reach",
			slot:     99,
			answer:   s.answer("preset.bin"),
			contains: []string{"slot 34A"},
		},
		{
			name:     "one read without a listing at all",
			slot:     0,
			answer:   s.answer("preset.bin"),
			listing:  errors.New("boom"),
			contains: []string{"slot 01A"},
		},
		{
			// Nothing guarantees what comes off a wire. Saying what arrived
			// beats printing a rig that would be wrong.
			name:     "an answer that is not a preset",
			slot:     0,
			answer:   map[any]any{1: 2},
			contains: []string{"map with 1 keys"},
		},
		{
			// A device answers an empty slot with no document at all. That
			// is a slot holding nothing rather than a failure, and a backup
			// has to know the difference.
			name:   "one the device answers with nothing",
			slot:   4,
			answer: nil,
			is:     slots.ErrEmptySlot,
		},
		{
			name: "one the device will not read",
			slot: 3,
			says: "slot 02A",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), tt.listing)

			if tt.says != "" {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, tt.slot).
					Return(nil, errors.New("boom"))
			} else {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, tt.slot).
					Return(tt.answer, nil)
			}

			var out bytes.Buffer

			err := slots.ShowWith(context.Background(), &out, s.dev,
				slots.DeviceOptions{Slot: tt.slot})

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				return
			}

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}
		})
	}
}

// TestShowWithKeepsTheAnswer covers the capture hook, which is how the wire
// format was read in the first place.
func (s *DevicePublicTestSuite) TestShowWithKeepsTheAnswer() {
	tests := []struct {
		name string
		path func() string
		err  bool
	}{
		{
			name: "somewhere it can write",
			path: func() string { return filepath.Join(s.T().TempDir(), "slot.bin") },
		},
		{
			name: "somewhere it cannot",
			path: func() string {
				return filepath.Join(s.T().TempDir(), "no", "such", "dir.bin")
			},
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := tt.path()
			s.T().Setenv("TONESTACK_USB_DUMP", path)

			s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
			s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).
				Return(s.answer("preset.bin"), nil)

			err := slots.ShowWith(context.Background(), &bytes.Buffer{}, s.dev,
				slots.DeviceOptions{Slot: 0})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			body, readErr := os.ReadFile(path) //nolint:gosec // a path this test wrote
			s.Require().NoError(readErr)
			s.Require().Equal(s.answer("preset.bin"), string(body), "kept verbatim")
		})
	}
}

// TestExportWith writes one slot out to a file.
func (s *DevicePublicTestSuite) TestExportWith() {
	tests := []struct {
		name     string
		opts     slots.ExportOptions
		answer   any
		out      string
		contains []string
		is       error
		err      bool
	}{
		{
			name:     "a rig, which is the default",
			opts:     slots.ExportOptions{Slot: 24},
			answer:   s.answer("switches.bin"),
			out:      "rig.yaml",
			contains: []string{"schema: RigSpec"},
		},
		{
			// A slot holding nothing leaves no file behind. An export used
			// to write a diagnostic into one and report success.
			name:   "a slot holding nothing",
			opts:   slots.ExportOptions{Slot: 4, As: "hlx"},
			answer: nil,
			out:    "empty.hlx",
			is:     slots.ErrEmptySlot,
		},
		{
			name:   "somewhere it cannot write",
			opts:   slots.ExportOptions{Slot: 0},
			answer: s.answer("preset.bin"),
			out:    filepath.Join("no", "such", "dir.yaml"),
			err:    true,
		},
		{
			name: "a slot the device will not read",
			opts: slots.ExportOptions{Slot: 0},
			out:  "x.yaml",
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := filepath.Join(s.T().TempDir(), tt.out)

			opts := tt.opts
			opts.OutputPath = path

			s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)

			if tt.err && tt.answer == nil {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, opts.Slot).
					Return(nil, errors.New("boom"))
			} else {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, opts.Slot).
					Return(tt.answer, nil)
			}

			var log bytes.Buffer

			err := slots.ExportWith(context.Background(), &log, s.dev, opts)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
				s.Require().NoFileExists(path)

				return
			}

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(log.String(), "wrote "+path)

			body, readErr := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(readErr)

			for _, want := range tt.contains {
				s.Require().Contains(string(body), want)
			}
		})
	}
}

// TestExportWithWritesTheDevicesOwnFile covers `--as hlx`, which the device
// path ignored: it wrote a rig whatever was asked for.
//
// The positions are the check that matters. A preset counts its blocks along
// the path and a device counts across a grid holding its routing too, so an
// export carrying the device's numbers is not the file HX Edit writes. Slot
// 27B exported from HX Edit holds these six at 1 to 6, and preset.bin is that
// same slot as the device sent it.
func (s *DevicePublicTestSuite) TestExportWithWritesTheDevicesOwnFile() {
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

// stand puts a session in place of the one that needs hardware, and takes it
// away again.
func (s *DevicePublicTestSuite) stand(dev sdk.Editor, err error) func() {
	restore := *slots.OpenDevice
	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return dev, err
	}

	return func() { *slots.OpenDevice = restore }
}

// TestTheCommandsThatFindTheirOwnDevice covers the three entry points, which
// are one line each: find a session, hand it on, release it. They are the
// only lines in the package that need hardware.
func (s *DevicePublicTestSuite) TestTheCommandsThatFindTheirOwnDevice() {
	out := filepath.Join(s.T().TempDir(), "rig.yaml")

	s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil).Times(3)
	// Twice for the two that read one slot, and once more for the listing,
	// which reads every named slot to say which of them hold anything.
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).
		Return(s.answer("preset.bin"), nil).Times(3)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 24).
		Return(s.answer("switches.bin"), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 79).
		Return(s.answer("empty.bin"), nil)
	s.dev.EXPECT().Close().Times(3)

	defer s.stand(s.dev, nil)()

	ctx := context.Background()

	s.Require().NoError(slots.ListDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}))
	s.Require().NoError(slots.ShowDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}))
	s.Require().NoError(slots.ExportDevice(ctx, &bytes.Buffer{},
		slots.ExportOptions{OutputPath: out}))
}

// TestShowDeviceOnAnEmptySlot covers the entry point somebody runs, where a
// slot holding nothing is an answer rather than a failure.
func (s *DevicePublicTestSuite) TestShowDeviceOnAnEmptySlot() {
	tests := []struct {
		name     string
		answer   any
		fails    error
		contains string
		says     string
	}{
		{
			name:     "a slot holding nothing",
			contains: "02B is empty",
		},
		{
			name:  "an error that is not an empty slot",
			fails: errors.New("no answer"),
			says:  "no answer",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
			s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 4).Return(tt.answer, tt.fails)
			s.dev.EXPECT().Close()

			defer s.stand(s.dev, nil)()

			var out bytes.Buffer

			err := slots.ShowDevice(context.Background(), &out,
				slots.DeviceOptions{Slot: 4})

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(out.String(), tt.contains)
		})
	}
}

// TestReportsADeviceItCannotOpen covers all three entry points finding none.
func (s *DevicePublicTestSuite) TestReportsADeviceItCannotOpen() {
	defer s.stand(nil, errors.New("no device found"))()

	ctx := context.Background()

	for _, err := range []error{
		slots.ListDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}),
		slots.ShowDevice(ctx, &bytes.Buffer{}, slots.DeviceOptions{}),
		slots.ExportDevice(ctx, &bytes.Buffer{}, slots.ExportOptions{}),
	} {
		s.Require().ErrorContains(err, "no device found")
	}
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
