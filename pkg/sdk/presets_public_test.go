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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// PresetsPublicTestSuite covers the operations a wrapper reaches through the
// Client, on both ends: a file, and the device it stands in for.
type PresetsPublicTestSuite struct {
	suite.Suite
}

func fixture(name string) string { return filepath.Join("testdata", name) }

// noDevice makes finding one fail, which is what every device path does on a
// machine with nothing plugged in.
//
// Enough to cover the branch. What happens once a session is open belongs to
// the flow's own tests, which have a scripted device to hand.
func (s *PresetsPublicTestSuite) noDevice() func() {
	restore := *sdk.OpenDevice
	*sdk.OpenDevice = func(context.Context) (device.Editor, error) {
		return nil, errors.New("no device found")
	}

	return func() { *sdk.OpenDevice = restore }
}

// where addresses the setlist fixture.
func (s *PresetsPublicTestSuite) where() sdk.Where {
	return sdk.Where{Path: fixture("setlist.hls"), CatalogPath: fixture("catalog.json")}
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
		got, err := sdk.New().Presets(context.Background(), s.where())

		s.Require().NoError(err)
		s.Require().NotEmpty(got.Slots)
	})

	s.Run("off a device that is not there", func() {
		defer s.noDevice()()

		_, err := sdk.New().Presets(context.Background(), sdk.Where{})
		s.Require().ErrorContains(err, "no device found")
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
			in: sdk.Read{
				Where: sdk.Where{CatalogPath: fixture("catalog.json")},
				File:  fixture("preset.hlx"),
			},
		},
		{name: "off a device that is not there", device: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.device {
				defer s.noDevice()()

				_, err := sdk.New().Preset(context.Background(), tt.in)
				s.Require().ErrorContains(err, "no device found")

				return
			}

			got, err := sdk.New().Preset(context.Background(), tt.in)

			s.Require().NoError(err)
			s.Require().False(got.Empty())
		})
	}
}

// TestExport covers writing one slot out.
func (s *PresetsPublicTestSuite) TestExport() {
	s.Run("out of a backup", func() {
		out := filepath.Join(s.T().TempDir(), "one.yaml")

		got, err := sdk.New().Export(context.Background(), sdk.Export{
			Where: s.where(), OutputPath: out,
		})

		s.Require().NoError(err)
		s.Require().Equal(out, got.Path)
	})

	s.Run("off a device that is not there", func() {
		defer s.noDevice()()

		_, err := sdk.New().Export(context.Background(), sdk.Export{
			OutputPath: filepath.Join(s.T().TempDir(), "one.yaml"),
		})
		s.Require().ErrorContains(err, "no device found")
	})
}

// TestImport covers putting a preset file into a slot.
func (s *PresetsPublicTestSuite) TestImport() {
	s.Run("into a backup", func() {
		out := filepath.Join(s.T().TempDir(), "out.hls")

		got, err := sdk.New().Import(context.Background(), sdk.Put{
			Where: s.where(), File: fixture("preset.hlx"),
			Slot: 1, OutputPath: out,
		})

		s.Require().NoError(err)
		s.Require().Equal(sdk.Imported, got.Action)
	})

	s.Run("onto a device that is not there", func() {
		defer s.noDevice()()

		_, err := sdk.New().Import(context.Background(), sdk.Put{
			File: fixture("preset.hlx"),
		})
		s.Require().ErrorContains(err, "no device found")
	})
}

// TestCopyAndSwap covers the two edits that move a preset between slots.
func (s *PresetsPublicTestSuite) TestCopyAndSwap() {
	tests := []struct {
		name string
		call func(sdk.Edit) (sdk.Change, error)
		want sdk.Action
	}{
		{
			name: "copy",
			call: func(in sdk.Edit) (sdk.Change, error) {
				return sdk.New().Copy(context.Background(), in)
			},
			want: sdk.Copied,
		},
		{
			name: "swap",
			call: func(in sdk.Edit) (sdk.Change, error) {
				return sdk.New().Swap(context.Background(), in)
			},
			want: sdk.Swapped,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name+" in a backup", func() {
			got, err := tt.call(sdk.Edit{
				Where:      s.where(),
				ToSlot:     1,
				OutputPath: filepath.Join(s.T().TempDir(), "out.hls"),
			})

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Action)
		})

		s.Run(tt.name+" on a device that is not there", func() {
			defer s.noDevice()()

			_, err := tt.call(sdk.Edit{ToSlot: 1})
			s.Require().ErrorContains(err, "no device found")
		})
	}
}

// TestSelect covers loading a preset, which writes nothing.
func (s *PresetsPublicTestSuite) TestSelect() {
	defer s.noDevice()()

	_, err := sdk.New().Select(context.Background(), sdk.Read{Slot: 4})
	s.Require().ErrorContains(err, "no device found")
}

// TestCompile covers building a preset from a rig on disk.
func (s *PresetsPublicTestSuite) TestCompile() {
	tests := []struct {
		name string
		rig  string
		err  bool
	}{
		{name: "a rig that is not there", rig: fixture("nope.yaml"), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := sdk.New().Compile(sdk.Compile{
				RigPath:     tt.rig,
				OutputPath:  filepath.Join(s.T().TempDir(), "out.hlx"),
				CatalogPath: fixture("catalog.json"),
			})

			s.Require().Error(err)
		})
	}
}

func TestPresetsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PresetsPublicTestSuite))
}
