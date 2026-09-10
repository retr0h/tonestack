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
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// EditDevicePublicTestSuite covers moving a preset between slots on a device.
//
// Nothing here reaches hardware. What it establishes is that the bytes leaving
// for the device are the bytes that arrived from it, because a preset is
// seeked through by a table of byte offsets and the surest way to keep those
// right is to change nothing.
type EditDevicePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	dev  *editable
}

// editable is a session that can both read and write.
type editable struct {
	*mocks.MockEditor
	*mocks.MockWriter
}

func (e *editable) Close() {}

func (s *EditDevicePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.dev = &editable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockWriter: mocks.NewMockWriter(s.ctrl),
	}
}

func (s *EditDevicePublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// answer returns one slot as the hardware sent it.
func (s *EditDevicePublicTestSuite) answer() []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

// listing is what the device says the setlist holds.
func (s *EditDevicePublicTestSuite) listing() []wire.Preset {
	return []wire.Preset{
		{Slot: 0, Name: "Chunky Monkey"},
		{Slot: 3, Name: "Black Rusty"},
	}
}

// backupDir returns somewhere a backup can go, or somewhere it cannot.
func (s *EditDevicePublicTestSuite) backupDir(bad bool) string {
	dir := s.T().TempDir()
	if !bad {
		return dir
	}

	// A file where a directory would have to be, so nothing can be made
	// under it.
	path := filepath.Join(dir, "in-the-way")
	s.Require().NoError(os.WriteFile(path, []byte("x"), 0o600))

	return filepath.Join(path, "under-it")
}

// expectListing sets up the listing every edit starts from.
func (s *EditDevicePublicTestSuite) expectListing(reader *mocks.MockEditor, ok bool) {
	if !ok {
		reader.EXPECT().Presets(gomock.Any(), 0).Return(nil, errors.New("boom"))

		return
	}

	reader.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
}

// expectRead sets up one slot being read: answered, refused, or answered with
// something that is not a preset.
func (s *EditDevicePublicTestSuite) expectRead(
	reader *mocks.MockEditor,
	slot int,
	outcome string,
) *gomock.Call {
	switch outcome {
	case "refused":
		return reader.EXPECT().ReadPreset(gomock.Any(), 0, slot).
			Return(nil, errors.New("boom"))
	case "not a preset":
		return reader.EXPECT().ReadPreset(gomock.Any(), 0, slot).
			Return(nil, nil)
	default:
		return reader.EXPECT().ReadPreset(gomock.Any(), 0, slot).
			Return(s.answer(), nil)
	}
}

// expectWrite sets up one slot being written, byte for byte. A preset that
// changed on the way through would leave the device's offset table pointing at
// the wrong places, and the device would accept it and then read the preset as
// empty.
func (s *EditDevicePublicTestSuite) expectWrite(
	slot int,
	name string,
	ok bool,
) *gomock.Call {
	call := s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, slot, name, s.answer())

	if !ok {
		return call.Return(errors.New("boom"))
	}

	return call.Return(nil)
}

// TestCopyWith writes one slot of a device over another.
func (s *EditDevicePublicTestSuite) TestCopyWith() {
	tests := []struct {
		name string
		// whether the listing comes back, how the read is answered, and
		// whether the write lands. An empty outcome means the call is never
		// reached.
		listed bool
		read   string
		write  string
		// a session that can read but not write.
		readOnly bool
		// how the destination read is answered. Empty means answered.
		readTo string
		// somewhere a backup cannot be written.
		badBackup bool
		// a writer nothing can be written to.

		contains []string
		errText  string
	}{
		{
			name:   "a preset sent back byte for byte",
			listed: true,
			read:   "answered",
			write:  "landed",
			contains: []string{
				"01A", "02A", "Chunky Monkey",
				// The destination is overwritten and there is no undo on a
				// device.
				"Black Rusty",
			},
		},
		{name: "a listing it cannot get", errText: "listing presets"},
		{
			// The destination is read so it can be kept. A device that will
			// not say what is there is a device that cannot be replaced
			// safely.
			name:    "a destination it cannot read",
			listed:  true,
			read:    "answered",
			readTo:  "refused",
			errText: "before replacing it",
		},
		{
			// Nowhere to put what the destination held, so it is not
			// replaced. A write nobody can undo does not happen.
			name:      "a backup it cannot write",
			listed:    true,
			read:      "answered",
			badBackup: true,
			errText:   "making room for a backup",
		},
		{
			name:    "a slot it cannot read",
			listed:  true,
			read:    "refused",
			errText: "reading slot 01A",
		},
		{
			name:    "an answer that is not a preset",
			listed:  true,
			read:    "not a preset",
			errText: "did not answer with a preset",
		},
		{
			name:    "a slot it cannot write",
			listed:  true,
			read:    "answered",
			write:   "refused",
			errText: "writing slot 02A",
		},
		{
			// Reading and writing are separate abilities, because writing is
			// the half that can destroy somebody's work.
			name:     "a session that cannot write",
			listed:   true,
			read:     "answered",
			readOnly: true,
			errText:  "cannot write",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reader := s.dev.MockEditor

			dev := device.Editor(s.dev)
			if tt.readOnly {
				reader = mocks.NewMockEditor(s.ctrl)
				dev = device.Editor(reader)
			}

			s.expectListing(reader, tt.listed)

			if tt.read != "" {
				s.expectRead(reader, 0, tt.read)
			}

			// The destination is read as well now, so that what it held can
			// be kept before it stops holding it. A swap needs no such read:
			// it has already read both slots to move them. A session that
			// cannot write never gets that far.
			if tt.read == "answered" && !tt.readOnly {
				outcome := tt.readTo
				if outcome == "" {
					outcome = "answered"
				}

				s.expectRead(reader, 3, outcome)
			}

			if tt.write != "" {
				s.expectWrite(3, "Chunky Monkey", tt.write == "landed")
			}

			change, err := slots.CopyWith(context.Background(), dev,
				slots.EditOptions{
					FromSlot: 0, ToSlot: 3, BackupDir: s.backupDir(tt.badBackup),
				})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(did(change), want)
			}
		})
	}
}

// TestSwapWith exchanges two slots on a device.
func (s *EditDevicePublicTestSuite) TestSwapWith() {
	tests := []struct {
		name string
		// the listing, then how each of the two slots answers, then whether
		// each of the two writes lands. An empty outcome means the call is
		// never reached.
		listed   bool
		reads    []string
		writes   []string
		readOnly bool
		// somewhere a backup cannot be written.
		badBackup bool

		contains string
		errText  string
	}{
		{
			name:     "both slots, each holding what the other did",
			listed:   true,
			reads:    []string{"answered", "answered"},
			writes:   []string{"landed", "landed"},
			contains: "swapped",
		},
		{name: "a listing it cannot get", errText: "listing presets"},
		{
			// A swap reads both slots to move them and keeps them from those
			// same reads, so nowhere to put them stops it.
			name:      "a backup it cannot write",
			listed:    true,
			reads:     []string{"answered", "answered"},
			badBackup: true,
			errText:   "making room for a backup",
		},
		{
			name:    "the first slot, which it cannot read",
			listed:  true,
			reads:   []string{"refused"},
			errText: "reading slot 01A",
		},
		{
			name:    "the second slot, which it cannot read",
			listed:  true,
			reads:   []string{"answered", "refused"},
			errText: "reading slot 02A",
		},
		{
			name:    "the destination, which it cannot write",
			listed:  true,
			reads:   []string{"answered", "answered"},
			writes:  []string{"refused"},
			errText: "writing slot 02A",
		},
		{
			name:    "the source, which it cannot write",
			listed:  true,
			reads:   []string{"answered", "answered"},
			writes:  []string{"landed", "refused"},
			errText: "writing slot 01A",
		},
		{
			name:     "a session that cannot write",
			listed:   true,
			reads:    []string{"answered", "answered"},
			readOnly: true,
			errText:  "cannot write",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reader := s.dev.MockEditor

			dev := device.Editor(s.dev)
			if tt.readOnly {
				reader = mocks.NewMockEditor(s.ctrl)
				dev = device.Editor(reader)
			}

			s.expectListing(reader, tt.listed)

			// A device that failed halfway through would leave one slot
			// holding a copy of the other and the original gone, so both
			// slots are read before either is written.
			var last *gomock.Call

			for i, outcome := range tt.reads {
				call := s.expectRead(reader, []int{0, 3}[i], outcome)
				if last != nil {
					call.After(last)
				}

				last = call
			}

			for i, outcome := range tt.writes {
				call := s.expectWrite(
					[]int{3, 0}[i],
					[]string{"Chunky Monkey", "Black Rusty"}[i],
					outcome == "landed")

				if last != nil {
					call.After(last)
				}
			}

			change, err := slots.SwapWith(context.Background(), dev,
				slots.EditOptions{
					FromSlot: 0, ToSlot: 3, BackupDir: s.backupDir(tt.badBackup),
				})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(did(change), tt.contains)
		})
	}
}

// TestCopyDeviceAndSwapDevice covers the two commands somebody actually runs.
// One line each: find a session, hand it on, release it.
func (s *EditDevicePublicTestSuite) TestCopyDeviceAndSwapDevice() {
	tests := []struct {
		name     string
		attached bool
	}{
		{name: "a device on the bus", attached: true},
		{name: "nothing on the bus"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := slots.OpenDevice
			defer func() { slots.OpenDevice = restore }()

			if !tt.attached {
				slots.OpenDevice = func(context.Context) (device.Editor, error) {
					return nil, errors.New("no device found")
				}
			} else {
				slots.OpenDevice = func(context.Context) (device.Editor, error) {
					return s.dev, nil
				}

				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil).Times(2)
				// Four: a copy reads its source and the destination it is
				// about to replace, a swap reads both of the slots it moves
				// and keeps them from those same reads.
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).
					Return(s.answer(), nil).Times(4)
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(
						gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(3)
			}

			ctx := context.Background()
			opts := slots.EditOptions{
				FromSlot: 0, ToSlot: 3, BackupDir: s.T().TempDir(),
			}

			if !tt.attached {
				_, copyErr := slots.CopyDevice(ctx, opts)
				_, swapErr := slots.SwapDevice(ctx, opts)
				s.Require().Error(copyErr)
				s.Require().Error(swapErr)

				return
			}

			_, err := slots.CopyDevice(ctx, opts)
			s.Require().NoError(err)

			_, err = slots.SwapDevice(ctx, opts)
			s.Require().NoError(err)
		})
	}
}

func TestEditDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(EditDevicePublicTestSuite))
}
