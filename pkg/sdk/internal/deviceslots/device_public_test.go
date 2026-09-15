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

package deviceslots_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	slotmocks "github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// flows are the operations, naming gear against the catalog at path. An empty
// path is the catalog built into this binary.
//
// The catalog comes from a generated double that opens the file whenever it is
// asked, which is what the sdk Client hands over after its first open.
func flows(
	t *testing.T,
	path string,
) *deviceslots.Flows {
	t.Helper()

	c := slotmocks.NewMockCatalogs(gomock.NewController(t))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.Open(path) },
	).AnyTimes()

	return &deviceslots.Flows{Catalogs: c}
}

// said renders a reading the way something displaying one would.
//
// The assertions here are about what was read, and what was read is a rig.
// Rendering it in the test rather than importing the one renderer keeps these
// operations free of anything that knows what a terminal is.
func said(
	t *testing.T,
	r result.Reading,
) string {
	t.Helper()

	if r.Answer != nil {
		return r.Answer.Shape
	}

	if r.Empty() {
		return "# " + r.Name + " is empty"
	}

	var buf bytes.Buffer

	require.NoError(t, rig.Write(&buf, r.Rig))

	return r.Name + "\n" + buf.String()
}

// formatFor is as, or a rig for a row that says nothing about the format. An
// export refuses the zero Format, so a row has to ask for one.
func formatFor(
	as result.Format,
) result.Format {
	if as == "" {
		return result.FormatRig
	}

	return as
}

// DevicePublicTestSuite covers reading a device, with no device attached.
//
// The answers are real: pkg/sdk/internal/wire/testdata holds three slots
// exactly as an HX Stomp handed them back: one full preset, one whose switches
// somebody labelled and coloured, and one empty slot.
type DevicePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	dev  *mocks.MockEditor
}

func (s *DevicePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.dev = mocks.NewMockEditor(s.ctrl)
	s.dev.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
}

func (s *DevicePublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// answer returns one slot as the hardware sent it.
func (s *DevicePublicTestSuite) answer(
	name string,
) []byte {
	raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", name))
	s.Require().NoError(err)

	return raw
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

// full is the reads a listing of one setlist makes: one per named slot,
// because a name says nothing about whether anything is in it.
func (s *DevicePublicTestSuite) full(
	setlist int,
) {
	s.dev.EXPECT().ReadPreset(gomock.Any(), setlist, 0).Return(s.answer("preset.bin"), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), setlist, 24).Return(s.answer("switches.bin"), nil)
	s.dev.EXPECT().ReadPreset(gomock.Any(), setlist, 79).Return(s.answer("empty.bin"), nil)
}

// TestList covers what one setlist on a device holds.
func (s *DevicePublicTestSuite) TestList() {
	tests := []struct {
		name    string
		setlist int
		catalog string
		// the device refuses to say what the setlist holds.
		listing error
		expect  func(setlist int)
		// what the slots must be called, and must not.
		contains []string
		absent   []string
		// how many hold a chain.
		used int
		says string
	}{
		{
			name:     "every slot holding a chain, and what is in it",
			expect:   s.full,
			contains: []string{"Chunky Monkey"},
			used:     2,
		},
		{
			// Two kinds of empty, and the difference is worth keeping: a
			// slot nobody named, and one that is named and holds nothing.
			// Both are answered; which to show is the renderer's.
			name:     "the ones holding nothing are answered too",
			expect:   s.full,
			contains: []string{"New Preset", "BAS:SVT Nrm"},
			used:     2,
		},
		{
			// Every read goes to the setlist asked about, not the first.
			name:     "a setlist other than the first",
			setlist:  2,
			expect:   s.full,
			contains: []string{"B15 Eras"},
			used:     2,
		},
		{name: "a catalog that will not open", catalog: "nowhere.json", says: "nowhere.json"},
		{
			// A catalog generated from another release should not hide every
			// slot behind the first model it cannot name.
			name:    "one whose model table this device has outgrown",
			catalog: filepath.Join("testdata", "unnamed.catalog.json"),
			expect:  s.full,
		},
		{
			name: "a slot the device will not read",
			expect: func(int) {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(nil, errors.New("boom"))
			},
			says: "boom",
		},
		{
			// One unreadable preset should not hide the hundred that read.
			name: "a preset that will not decode",
			expect: func(int) {
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return([]byte("not a preset"), nil)
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 24).
					Return(nil, &device.NotAPresetError{Result: 42})
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, 79).Return(nil, nil)
			},
		},
		{
			// The one call the listing cannot do without.
			name:    "a listing it cannot get",
			listing: errors.New("boom"),
			says:    "listing presets",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.dev.EXPECT().Presets(gomock.Any(), tt.setlist).Return(s.listing(), tt.listing)

			if tt.expect != nil {
				tt.expect(tt.setlist)
			}

			listing, err := flows(
				s.T(),
				tt.catalog,
			).List(context.Background(), s.dev, tt.setlist)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal("HX Stomp", listing.Name)
			s.Require().Equal(tt.used, listing.Used())

			named := make([]string, 0, len(listing.Slots))
			for _, h := range listing.Slots {
				named = append(named, h.Name)
			}

			for _, want := range tt.contains {
				s.Require().Contains(named, want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(named, gone)
			}
		})
	}
}

// TestShow covers reading one slot off a device, and keeping what it
// answered.
func (s *DevicePublicTestSuite) TestShow() {
	tests := []struct {
		name string
		at   slotpkg.Address
		// what the device answered with: a preset, something that is not
		// one, or a refusal.
		answer    []byte
		notPreset any
		unread    bool
		listing   error
		// where the answer is captured: nowhere, somewhere it can be
		// written, or somewhere that refuses.
		capture string

		contains []string
		// the capture holds the answer byte for byte.
		keptVerbatim bool
		// the capture holds a decoded answer as JSON.
		keptJSON bool
		is       error
		says     string
	}{
		{
			name:   "a preset, as a rig",
			at:     slotpkg.Address{Slot: 24},
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
			// The setlist half of the address is read, and the name comes from
			// that setlist's listing.
			name:     "a preset in another setlist",
			at:       slotpkg.Address{Setlist: 1, Slot: 24},
			answer:   s.answer("switches.bin"),
			contains: []string{"name: B15 Eras"},
		},
		{
			name:     "one holding no blocks",
			at:       slotpkg.Address{Slot: 1},
			answer:   s.answer("empty.bin"),
			contains: []string{"is empty"},
		},
		{
			// The name is a convenience. A slot beyond what the listing
			// returned is still read, and named by where it sits.
			name:     "one the listing does not reach",
			at:       slotpkg.Address{Slot: 99},
			answer:   s.answer("preset.bin"),
			contains: []string{"slot 34A"},
		},
		{
			name:     "one read without a listing at all",
			answer:   s.answer("preset.bin"),
			listing:  errors.New("boom"),
			contains: []string{"slot 01A"},
		},
		{
			// Nothing guarantees what comes off a wire. Saying what arrived
			// beats printing a rig that would be wrong.
			name:      "an answer that is not a preset",
			notPreset: map[any]any{1: 2},
			contains:  []string{"map with 1 keys"},
		},
		{
			// A device answers an empty slot with no document at all. That
			// is a slot holding nothing rather than a failure, and a backup
			// has to know the difference.
			name: "one the device answers with nothing",
			at:   slotpkg.Address{Slot: 4},
			is:   deviceslots.ErrEmptySlot,
		},
		{
			name:   "one the device will not read",
			at:     slotpkg.Address{Slot: 3},
			unread: true,
			says:   "slot 02A",
		},
		{
			// How the wire format was read in the first place.
			name:         "an answer kept where it can be written",
			answer:       s.answer("preset.bin"),
			capture:      "kept",
			keptVerbatim: true,
		},
		{
			name:    "an answer with nowhere to keep it",
			answer:  s.answer("preset.bin"),
			capture: "refused",
			says:    "capturing",
		},
		{
			// A decoded document is kept as JSON, since there are no
			// original bytes to keep.
			name:      "an answer nobody could decode, kept",
			notPreset: map[any]any{1: 2},
			capture:   "kept",
			keptJSON:  true,
		},
		{
			name:      "an answer nobody could decode, with nowhere to keep it",
			notPreset: map[any]any{1: 2},
			capture:   "refused",
			says:      "capturing",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.dev.EXPECT().Presets(gomock.Any(), tt.at.Setlist).Return(s.listing(), tt.listing)

			read := s.dev.EXPECT().ReadPreset(gomock.Any(), tt.at.Setlist, tt.at.Slot)

			switch {
			case tt.unread:
				read.Return(nil, errors.New("boom"))
			case tt.notPreset != nil:
				read.Return(nil, &device.NotAPresetError{Result: tt.notPreset})
			default:
				read.Return(tt.answer, nil)
			}

			var kept bytes.Buffer

			f := &deviceslots.Flows{}

			switch tt.capture {
			case "kept":
				f.Capture = &kept
			case "refused":
				f.Capture = refusing{}
			}

			got, err := f.Show(context.Background(), s.dev, tt.at)

			switch {
			case tt.is != nil:
				s.Require().ErrorIs(err, tt.is)

				return
			case tt.says != "":
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(said(s.T(), got), want)
			}

			switch {
			case tt.keptVerbatim:
				s.Require().Equal(tt.answer, kept.Bytes(), "kept verbatim")
			case tt.keptJSON:
				s.Require().Contains(kept.String(), "1")
			default:
				s.Require().Zero(kept.Len(), "nobody asked for it to be kept")
			}
		})
	}
}

// TestExport covers writing one slot off a device out to a file.
func (s *DevicePublicTestSuite) TestExport() {
	tests := []struct {
		name     string
		slot     int
		as       result.Format
		answer   []byte
		out      string
		contains []string
		// a translator that hands back a document that will not encode.
		unencodable bool
		// the format is refused before the device is asked anything.
		refused bool
		// the file must hold the blocks where HX Edit puts them.
		laidOut bool
		is      error
		err     bool
		errText string
	}{
		{
			name:     "a rig, which is the default",
			slot:     24,
			answer:   s.answer("switches.bin"),
			out:      "rig.yaml",
			contains: []string{"schema: RigSpec"},
		},
		{
			// The device path once ignored the format and wrote a rig
			// whatever was asked for.
			name:    "the device's own file, laid out as HX Edit writes it",
			as:      result.FormatPreset,
			answer:  s.answer("preset.bin"),
			out:     "slot.hlx",
			laidOut: true,
		},
		{
			// A slot holding nothing leaves no file behind. An export used
			// to write a diagnostic into one and report success.
			name: "a slot holding nothing",
			slot: 4,
			as:   result.FormatPreset,
			out:  "empty.hlx",
			is:   deviceslots.ErrEmptySlot,
		},
		{
			// A slot the device answers for with nothing on the grid. As the
			// device's own file there is no document to write, and a file
			// saying null is not a preset.
			name:   "a slot with no blocks",
			slot:   79,
			as:     result.FormatPreset,
			answer: s.answer("empty.bin"),
			out:    "blocks.hlx",
			is:     deviceslots.ErrEmptySlot,
		},
		{
			// Reported, rather than a file holding nothing where a preset
			// was meant to be.
			name:        "a preset that will not encode",
			as:          result.FormatPreset,
			answer:      s.answer("preset.bin"),
			out:         "unencodable.hlx",
			unencodable: true,
			err:         true,
			errText:     "encoding preset",
		},
		{
			name:   "somewhere it cannot write",
			answer: s.answer("preset.bin"),
			out:    filepath.Join("no", "such", "dir.yaml"),
			err:    true,
		},
		{name: "a slot the device will not read", out: "x.yaml", err: true},
		{
			// No expectation is set on the device, so asking it anything
			// fails the row.
			name:    "a format that is neither a rig nor the device's own file",
			as:      result.Format("yaml"),
			out:     "x.yaml",
			refused: true,
			err:     true,
			errText: result.ErrUnknownFormat.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := filepath.Join(s.T().TempDir(), tt.out)

			switch {
			case tt.refused:
			case tt.err && tt.answer == nil:
				s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, tt.slot).Return(nil, errors.New("boom"))
			default:
				s.dev.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
				s.dev.EXPECT().ReadPreset(gomock.Any(), 0, tt.slot).Return(tt.answer, nil)
			}

			f := &deviceslots.Flows{}

			if tt.unencodable {
				translator := slotmocks.NewMockTranslator(s.ctrl)
				translator.EXPECT().
					Document(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&preset.Document{Meta: json.RawMessage("{")}, false, nil)

				f.Translator = translator
			}

			written, err := f.Export(context.Background(), s.dev,
				slotpkg.Address{Slot: tt.slot}, path, formatFor(tt.as))

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
				s.Require().NoFileExists(path)

				return
			}

			if tt.err {
				s.Require().Error(err)
				s.Require().NoFileExists(path)

				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(path, written.Path)

			body, readErr := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(readErr)

			for _, want := range tt.contains {
				s.Require().Contains(string(body), want)
			}

			if tt.laidOut {
				s.requireLaidOut(body)
			}
		})
	}
}

// requireLaidOut checks a written preset holds its blocks where HX Edit puts
// them.
//
// The positions are the check that matters. A preset counts its blocks along
// the path and a device counts across a grid holding its routing too, so an
// export carrying the device's numbers is not the file HX Edit writes. Slot
// 27B exported from HX Edit holds these six at 1 to 6, and preset.bin is that
// same slot as the device sent it.
func (s *DevicePublicTestSuite) requireLaidOut(
	body []byte,
) {
	// A tone holds more than blocks: the globals, the snapshots, the routing.
	// So entries are read one at a time rather than through one shape.
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

// refusing is a capture that will not take a write.
//
// Written by hand because io.Writer is the standard library's interface.
type refusing struct{}

func (refusing) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("nowhere to keep it")
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
