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

package sdk_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// PresetsPublicTestSuite covers the operations a wrapper reaches through the
// Client, on both ends: a file, and the device it stands in for.
type PresetsPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *PresetsPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

func fixture(name string) string { return filepath.Join("testdata", name) }

// client is a Client naming gear against the catalog fixture, over a bus that
// finds nothing, which is what every device path does on a machine with
// nothing plugged in.
//
// Enough to cover the branch. What happens once a session is open belongs to
// the flow's own tests, which have a scripted device to hand.
func (s *PresetsPublicTestSuite) client() *sdk.Client {
	b := mocks.NewMockOpener(s.ctrl)
	b.EXPECT().Open(gomock.Any()).Return(nil, errors.New("no device found")).AnyTimes()

	return sdk.New(sdk.WithCatalog(fixture("catalog.json")), sdk.WithDevices(b))
}

// attachedTo is a Client over a pedal that holds a real preset in every slot,
// takes every write and every switch, and is let go once per call.
func (s *PresetsPublicTestSuite) attachedTo() *sdk.Client {
	body, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	dev := &attached{
		MockEditor:   mocks.NewMockEditor(s.ctrl),
		MockWriter:   mocks.NewMockWriter(s.ctrl),
		MockSelector: mocks.NewMockSelector(s.ctrl),
	}
	dev.MockEditor.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil).AnyTimes()
	dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).Return(body, nil).AnyTimes()
	dev.MockEditor.EXPECT().Close().Return(nil)
	dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
	dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, gomock.Any()).Return(nil).AnyTimes()

	bus := mocks.NewMockOpener(s.ctrl)
	bus.EXPECT().Open(gomock.Any()).Return(dev, nil)

	return sdk.New(sdk.WithDevices(bus), sdk.WithBackupDir(s.T().TempDir()))
}

// refusing is a Client whose bus fails the test if a call reaches for a device
// at all, which is what an input refused before it is opened must never do.
func (s *PresetsPublicTestSuite) refusing() *sdk.Client {
	// No expectations: gomock fails the test on any call.
	return sdk.New(sdk.WithDevices(mocks.NewMockOpener(s.ctrl)))
}

// where addresses the setlist fixture.
func (s *PresetsPublicTestSuite) where() sdk.Where {
	return sdk.Where{Path: fixture("setlist.hls")}
}

// TestOnDevice covers telling the two ends apart.
func (s *PresetsPublicTestSuite) TestOnDevice() {
	tests := []struct {
		name string
		in   sdk.Where
		want bool
	}{
		{
			// No path means the attached device, which is what somebody
			// with one plugged in almost always wants.
			name: "nothing said about where",
			want: true,
		},
		{name: "a backup on disk", in: sdk.Where{Path: "setlist.hls"}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.OnDevice())
		})
	}
}

// TestPresets covers reading what a setlist holds.
func (s *PresetsPublicTestSuite) TestPresets() {
	s.Run("out of a backup", func() {
		got, err := s.client().Presets(context.Background(), s.where())

		s.Require().NoError(err)
		s.Require().NotEmpty(got.Slots)
	})

	s.Run("off a device that is not there", func() {
		_, err := s.client().Presets(context.Background(), sdk.Where{})
		s.Require().ErrorContains(err, "no device found")
	})

	s.Run("off a device, in a Session of its own", func() {
		got, err := s.attachedTo().Presets(context.Background(), sdk.Where{})
		s.Require().NoError(err)
		s.Require().Len(got.Slots, 2)
	})

	s.Run("off a device, when the flow panics", func() {
		dev := &attached{
			MockEditor:   mocks.NewMockEditor(s.ctrl),
			MockWriter:   mocks.NewMockWriter(s.ctrl),
			MockSelector: mocks.NewMockSelector(s.ctrl),
		}
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).DoAndReturn(
			func(context.Context, int) ([]wire.Preset, error) {
				panic("a bug inside a flow")
			})
		dev.MockEditor.EXPECT().Close().Return(nil).Times(2)

		bus := mocks.NewMockOpener(s.ctrl)
		bus.EXPECT().Open(gomock.Any()).Return(dev, nil).Times(2)

		client := sdk.New(sdk.WithDevices(bus))

		s.Require().Panics(func() {
			_, _ = client.Presets(context.Background(), sdk.Where{})
		})

		// The pedal was let go on the way up, so the Client opens it again
		// rather than waiting on a claim nobody will give back.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		again, err := client.Open(ctx)
		s.Require().NoError(err)
		s.Require().NoError(again.Close())
	})
}

// TestPreset covers reading one slot as a rig.
func (s *PresetsPublicTestSuite) TestPreset() {
	tests := []struct {
		name   string
		in     sdk.Read
		device bool
	}{
		{name: "a slot in a backup", in: sdk.Read{Where: s.where()}},
		{
			// A standalone preset is neither a device nor a setlist, so it
			// is asked for by name and answered before either.
			name: "a preset in a file of its own",
			in:   sdk.Read{File: fixture("preset.hlx")},
		},
		{name: "off a device that is not there", device: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.client().Preset(context.Background(), tt.in)

			if tt.device {
				s.Require().ErrorContains(err, "no device found")

				return
			}

			s.Require().NoError(err)
			s.Require().False(got.Empty())
		})
	}
}

// TestExport covers writing one slot out.
func (s *PresetsPublicTestSuite) TestExport() {
	s.Run("out of a backup", func() {
		out := filepath.Join(s.T().TempDir(), "one.yaml")

		got, err := s.client().Export(context.Background(), sdk.Export{
			Where: s.where(), OutputPath: out,
		})

		s.Require().NoError(err)
		s.Require().Equal(out, got.Path)
	})

	s.Run("off a device that is not there", func() {
		_, err := s.client().Export(context.Background(), sdk.Export{
			OutputPath: filepath.Join(s.T().TempDir(), "one.yaml"),
		})
		s.Require().ErrorContains(err, "no device found")
	})

	s.Run("off a device, in a Session of its own", func() {
		out := filepath.Join(s.T().TempDir(), "one.yaml")

		got, err := s.attachedTo().Export(context.Background(), sdk.Export{OutputPath: out})
		s.Require().NoError(err)
		s.Require().Equal(out, got.Path)
	})
}

// TestImport covers putting a preset file into a slot.
func (s *PresetsPublicTestSuite) TestImport() {
	s.Run("into a backup", func() {
		out := filepath.Join(s.T().TempDir(), "out.hls")

		got, err := s.client().Import(context.Background(), sdk.Put{
			Where: s.where(), File: fixture("preset.hlx"),
			Slot: 1, OutputPath: out,
		})

		s.Require().NoError(err)
		s.Require().Equal(sdk.Imported, got.Action)
	})

	s.Run("onto a device that is not there", func() {
		_, err := s.client().Import(context.Background(), sdk.Put{
			File: fixture("preset.hlx"),
		})
		s.Require().ErrorContains(err, "no device found")
	})

	s.Run("onto a device, in a Session of its own", func() {
		got, err := s.attachedTo().Import(context.Background(), sdk.Put{
			File: filepath.Join("internal", "compile", "testdata", "preset0.hlx"),
			Slot: 1,
		})
		s.Require().NoError(err)
		s.Require().Equal(sdk.Imported, got.Action)
	})
}

// TestCopyAndSwap covers the two edits that move a preset between slots.
func (s *PresetsPublicTestSuite) TestCopyAndSwap() {
	tests := []struct {
		name string
		call func(*sdk.Client, sdk.Edit) (sdk.Change, error)
		want sdk.Action
	}{
		{
			name: "copy",
			call: func(c *sdk.Client, in sdk.Edit) (sdk.Change, error) {
				return c.Copy(context.Background(), in)
			},
			want: sdk.Copied,
		},
		{
			name: "swap",
			call: func(c *sdk.Client, in sdk.Edit) (sdk.Change, error) {
				return c.Swap(context.Background(), in)
			},
			want: sdk.Swapped,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name+" in a backup", func() {
			got, err := tt.call(s.client(), sdk.Edit{
				Where:      s.where(),
				ToSlot:     1,
				OutputPath: filepath.Join(s.T().TempDir(), "out.hls"),
			})

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Action)
		})

		s.Run(tt.name+" on a device that is not there", func() {
			_, err := tt.call(s.client(), sdk.Edit{ToSlot: 1})
			s.Require().ErrorContains(err, "no device found")
		})

		s.Run(tt.name+" on a device, in a Session of its own", func() {
			got, err := tt.call(s.attachedTo(), sdk.Edit{ToSlot: 1})
			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Action)
		})

		s.Run(tt.name+" names a setlist on Where", func() {
			// Edit already has FromSetlist and ToSetlist, one per side of
			// the move. Where.Setlist has no side to belong to and would
			// otherwise be silently ignored.
			_, err := tt.call(s.refusing(), sdk.Edit{Where: sdk.Where{Setlist: 2}, ToSlot: 1})
			s.Require().ErrorIs(err, sdk.ErrEditSetlist)
		})
	}
}

// TestSelect covers loading a preset, which writes nothing.
func (s *PresetsPublicTestSuite) TestSelect() {
	s.Run("off a device that is not there", func() {
		_, err := s.client().Select(context.Background(), sdk.Read{Slot: 4})
		s.Require().ErrorContains(err, "no device found")
	})

	s.Run("on a device, in a Session of its own", func() {
		got, err := s.attachedTo().Select(context.Background(), sdk.Read{Slot: 1})
		s.Require().NoError(err)
		s.Require().Equal(sdk.Selected, got.Action)
	})

	// A slot is only ever selected on the device that plays it. Naming a
	// file either way must be refused before a device is even sought.
	tests := []struct {
		name string
		in   sdk.Read
	}{
		{
			name: "Where names a file",
			in:   sdk.Read{Where: sdk.Where{Path: fixture("setlist.hls")}, Slot: 4},
		},
		{
			name: "File names a standalone preset",
			in:   sdk.Read{File: fixture("preset.hlx"), Slot: 4},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := s.refusing().Select(context.Background(), tt.in)
			s.Require().ErrorIs(err, sdk.ErrSelectNeedsDevice)
		})
	}
}

// TestCompile covers building a preset from a rig on disk.
func (s *PresetsPublicTestSuite) TestCompile() {
	tests := []struct {
		name string
		ctx  context.Context
		rig  string
	}{
		{name: "a rig that is not there", rig: fixture("nope.yaml")},
		{name: "a caller who stopped waiting", ctx: cancelled(), rig: fixture("nope.yaml")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			_, err := s.client().Compile(ctx, sdk.Compile{
				RigPath:    tt.rig,
				OutputPath: filepath.Join(s.T().TempDir(), "out.hlx"),
			})

			s.Require().Error(err)
		})
	}
}

func TestPresetsPublicTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(PresetsPublicTestSuite))
}
