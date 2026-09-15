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
	"io/fs"
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
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// PresetsPublicTestSuite covers the Client's one-shot device methods, a
// standalone preset, and compiling a rig.
type PresetsPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *PresetsPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
}

func fixture(
	name string,
) string {
	return filepath.Join("testdata", name)
}

// absent is a Client naming gear against the catalog fixture, over a bus that
// finds nothing, which is what every device method meets on a machine with
// nothing plugged in.
//
// Enough to cover the branch. What happens once a session is open belongs to
// the flow's own tests, which have a scripted device to hand.
func (s *PresetsPublicTestSuite) absent() *sdk.Client {
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

// TestPresets covers listing a setlist on the device.
func (s *PresetsPublicTestSuite) TestPresets() {
	tests := []struct {
		name   string
		client func() *sdk.Client
		slots  int
		says   string
	}{
		{name: "a device that is not there", client: s.absent, says: "no device found"},
		{name: "a device, in a Session of its own", client: s.attachedTo, slots: 2},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := tt.client().Presets(context.Background(), 0)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got.Slots, tt.slots)
		})
	}

	s.Run("a device, when the flow panics", func() {
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
			_, _ = client.Presets(context.Background(), 0)
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

// TestPreset covers reading one slot on the device as a rig.
func (s *PresetsPublicTestSuite) TestPreset() {
	tests := []struct {
		name   string
		client func() *sdk.Client
		named  string
		says   string
	}{
		{name: "a device that is not there", client: s.absent, says: "no device found"},
		{name: "a device, in a Session of its own", client: s.attachedTo, named: "Longview"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := tt.client().Preset(context.Background(), slot.Address{Slot: 1})

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.named, got.Name)
			s.Require().False(got.Empty())
		})
	}
}

// TestPresetFile covers reading a standalone .hlx.
func (s *PresetsPublicTestSuite) TestPresetFile() {
	tests := []struct {
		name string
		ctx  context.Context
		path string
		says string
	}{
		{
			// A standalone preset is neither a device nor a setlist, so it
			// needs no hardware and no address.
			name: "a preset in a file of its own",
			path: fixture("preset.hlx"),
		},
		{name: "a preset that is not there", path: fixture("nope.hlx"), says: "opening"},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			path: fixture("preset.hlx"),
			says: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			// No expectations on the bus: reading a file must not reach it.
			client := sdk.New(sdk.WithCatalog(fixture("catalog.json")),
				sdk.WithDevices(mocks.NewMockOpener(s.ctrl)))

			got, err := client.PresetFile(ctx, tt.path)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().False(got.Empty())
		})
	}
}

// TestExport covers writing one slot on the device out to a file.
func (s *PresetsPublicTestSuite) TestExport() {
	tests := []struct {
		name   string
		client func() *sdk.Client
		as     sdk.Format
		file   string
		says   string
	}{
		{
			name:   "a device that is not there",
			client: s.absent,
			file:   "one.yaml",
			says:   "no device found",
		},
		{
			name:   "a rig, in a Session of its own",
			client: s.attachedTo,
			as:     sdk.FormatRig,
			file:   "one.yaml",
		},
		{
			name:   "the device's own file, in a Session of its own",
			client: s.attachedTo,
			as:     sdk.FormatPreset,
			file:   "one.hlx",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), tt.file)

			got, err := tt.client().
				Export(context.Background(), slot.Address{}, out, tt.as, sdk.ReplaceExisting)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().FileExists(out)
		})
	}
}

// TestImport covers putting a preset file into a slot on the device.
func (s *PresetsPublicTestSuite) TestImport() {
	tests := []struct {
		name   string
		client func() *sdk.Client
		says   string
	}{
		{name: "a device that is not there", client: s.absent, says: "no device found"},
		{name: "a device, in a Session of its own", client: s.attachedTo},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := tt.client().Import(context.Background(),
				filepath.Join("internal", "compile", "testdata", "preset0.hlx"),
				slot.Address{Slot: 1})

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(sdk.Imported, got.Action)
		})
	}
}

// TestCopyAndSwap covers the two edits that move a preset between slots on
// the device.
func (s *PresetsPublicTestSuite) TestCopyAndSwap() {
	tests := []struct {
		name   string
		client func() *sdk.Client
		call   func(*sdk.Client, slot.Address, slot.Address) (sdk.Change, error)
		want   sdk.Action
		says   string
	}{
		{
			name:   "copy on a device that is not there",
			client: s.absent,
			call:   copyOn,
			says:   "no device found",
		},
		{
			name:   "copy on a device, in a Session of its own",
			client: s.attachedTo,
			call:   copyOn,
			want:   sdk.Copied,
		},
		{
			name:   "swap on a device that is not there",
			client: s.absent,
			call:   swapOn,
			says:   "no device found",
		},
		{
			name:   "swap on a device, in a Session of its own",
			client: s.attachedTo,
			call:   swapOn,
			want:   sdk.Swapped,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := tt.call(tt.client(), slot.Address{}, slot.Address{Slot: 1})

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Action)
		})
	}
}

// copyOn copies one slot on the device over another.
func copyOn(
	c *sdk.Client,
	from, to slot.Address,
) (sdk.Change, error) {
	return c.Copy(context.Background(), from, to)
}

// swapOn exchanges two slots on the device.
func swapOn(
	c *sdk.Client,
	a, b slot.Address,
) (sdk.Change, error) {
	return c.Swap(context.Background(), a, b)
}

// TestSelect covers loading a preset, which writes nothing.
func (s *PresetsPublicTestSuite) TestSelect() {
	tests := []struct {
		name   string
		client func() *sdk.Client
		says   string
	}{
		{name: "a device that is not there", client: s.absent, says: "no device found"},
		{name: "a device, in a Session of its own", client: s.attachedTo},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := tt.client().Select(context.Background(), slot.Address{Slot: 1})

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(sdk.Selected, got.Action)
		})
	}
}

// rigFile writes a rig somebody typed, about subject, and returns its path.
func (s *PresetsPublicTestSuite) rigFile(
	dir string,
	subject string,
) string {
	path := filepath.Join(dir, subject+".yaml")

	s.Require().NoError(os.WriteFile(path, []byte(`schema: RigSpec
version: 2
id: typed
subject: { kind: sound, name: `+subject+` }
instrument: bass
chain:
  - { role: amp, gear: Ampeg SVT }
`), 0o600))

	return path
}

// compiled is what a compile answered and the preset it wrote.
type compiled struct {
	built sdk.Built
	body  string
}

// TestCompile covers building a preset from a rig on disk.
//
// Compile is the one input struct left, so each of its fields has a row of its
// own that changes that field alone and shows the result changing with it. A
// field no path reads would leave its row identical to the baseline.
func (s *PresetsPublicTestSuite) TestCompile() {
	dir := s.T().TempDir()
	typed := s.rigFile(dir, "Typed")
	other := s.rigFile(dir, "Other")

	compile := func(ctx context.Context, in sdk.Compile) (compiled, error) {
		built, err := sdk.New().Compile(ctx, in)
		if err != nil {
			return compiled{}, err
		}

		body, err := os.ReadFile(in.Out) //nolint:gosec // a path this test chose
		s.Require().NoError(err)

		return compiled{built: built, body: string(body)}, nil
	}

	base, err := compile(context.Background(), sdk.Compile{
		Rig: typed,
		Out: filepath.Join(dir, "base.hlx"),
	})
	s.Require().NoError(err)

	tests := []struct {
		name  string
		ctx   context.Context
		in    sdk.Compile
		check func(got compiled)
		says  string
		is    error
	}{
		{
			// base.hlx is already there, so keeping it refuses the write and
			// leaves the preset the baseline wrote.
			name: "Existing decides what happens to a file already there",
			in: sdk.Compile{
				Rig:      other,
				Out:      filepath.Join(dir, "base.hlx"),
				Existing: sdk.KeepExisting,
			},
			is: fs.ErrExist,
		},
		{
			name: "Rig decides what is built",
			in:   sdk.Compile{Rig: other, Out: filepath.Join(dir, "rig.hlx")},
			check: func(got compiled) {
				s.Require().Equal("Other", got.built.Name)
				s.Require().NotEqual(base.built.Name, got.built.Name)
			},
		},
		{
			// A template's routing and controllers are kept around a rig
			// nobody lifted, so the file differs while the rig does not.
			name: "Template decides what the chain is written into",
			in: sdk.Compile{
				Rig:      typed,
				Template: fixture("preset.hlx"),
				Out:      filepath.Join(dir, "template.hlx"),
			},
			check: func(got compiled) {
				s.Require().Equal(base.built.Name, got.built.Name)
				s.Require().NotEqual(base.body, got.body)
			},
		},
		{
			name: "Out decides where the preset goes",
			in:   sdk.Compile{Rig: typed, Out: filepath.Join(dir, "elsewhere.hlx")},
			check: func(got compiled) {
				s.Require().Equal(filepath.Join(dir, "elsewhere.hlx"), got.built.Path)
				s.Require().NotEqual(base.built.Path, got.built.Path)
				s.Require().Equal(base.body, got.body, "and nothing about what is in it")
			},
		},
		{
			name: "a rig that is not there",
			in:   sdk.Compile{Rig: fixture("nope.yaml"), Out: filepath.Join(dir, "nope.hlx")},
			says: "opening",
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			in:   sdk.Compile{Rig: typed, Out: filepath.Join(dir, "cancelled.hlx")},
			says: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := compile(ctx, tt.in)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				body, readErr := os.ReadFile(tt.in.Out) //nolint:gosec // a path this test chose
				s.Require().NoError(readErr)
				s.Require().Equal(base.body, string(body))

				return
			}

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)
				s.Require().NoFileExists(tt.in.Out)

				return
			}

			s.Require().NoError(err)
			tt.check(got)
		})
	}
}

func TestPresetsPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(PresetsPublicTestSuite))
}
