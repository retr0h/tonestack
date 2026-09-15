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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// SetlistPublicTestSuite covers a .hls or .hlb on disk, which needs no
// hardware.
type SetlistPublicTestSuite struct {
	suite.Suite
}

// setlist is the setlist fixture at path, read through a Client whose bus
// fails the test if anything reaches for a device.
func (s *SetlistPublicTestSuite) setlist(
	path string,
) *sdk.Setlist {
	return sdk.New(
		sdk.WithCatalog(fixture("catalog.json")),
		sdk.WithDevices(mocks.NewMockOpener(gomock.NewController(s.T()))),
	).Setlist(path)
}

// TestSetlist covers addressing a file.
func (s *SetlistPublicTestSuite) TestSetlist() {
	s.Run("a file that is not there, which is not read until asked", func() {
		s.Require().NotNil(sdk.New().Setlist(fixture("nope.hls")))
	})
}

// TestPresets covers what one setlist in a file holds.
func (s *SetlistPublicTestSuite) TestPresets() {
	tests := []struct {
		name    string
		ctx     context.Context
		path    string
		setlist int
		says    string
	}{
		{name: "a backup on disk", path: fixture("setlist.hls")},
		{name: "a file that is not there", path: fixture("nope.hls"), says: "opening"},
		{
			name:    "a setlist the file does not hold",
			path:    fixture("setlist.hls"),
			setlist: 9,
			says:    "no such slot",
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			path: fixture("setlist.hls"),
			says: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.setlist(tt.path).Presets(orBackground(tt.ctx), tt.setlist)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(got.Slots)
		})
	}
}

// TestPreset covers reading one slot of a file as a rig.
func (s *SetlistPublicTestSuite) TestPreset() {
	tests := []struct {
		name string
		path string
		at   slot.Address
		says string
	}{
		{name: "a slot in a backup", path: fixture("setlist.hls")},
		{
			name: "a slot the file does not hold",
			path: fixture("setlist.hls"),
			at:   slot.Address{Slot: 99},
			says: "no such slot",
		},
		{name: "a file that is not there", path: fixture("nope.hls"), says: "opening"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.setlist(tt.path).Preset(context.Background(), tt.at)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().False(got.Empty())
		})
	}
}

// TestExport covers writing one slot of a file out to a file of its own.
func (s *SetlistPublicTestSuite) TestExport() {
	tests := []struct {
		name string
		path string
		as   sdk.Format
		file string
		says string
	}{
		{name: "a rig", path: fixture("setlist.hls"), as: sdk.FormatRig, file: "one.yaml"},
		{
			name: "the device's own file",
			path: fixture("setlist.hls"),
			as:   sdk.FormatPreset,
			file: "one.hlx",
		},
		{
			name: "out of a file that is not there",
			path: fixture("nope.hls"),
			file: "one.yaml",
			says: "opening",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), tt.file)

			got, err := s.setlist(tt.path).Export(context.Background(), slot.Address{}, out, tt.as)

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

// TestImport covers putting a preset file into one slot of a file.
func (s *SetlistPublicTestSuite) TestImport() {
	tests := []struct {
		name string
		file string
		says string
	}{
		{name: "a preset into a slot", file: fixture("preset.hlx")},
		{name: "a preset that is not there", file: fixture("nope.hlx"), says: "opening"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "out.hls")

			got, err := s.setlist(fixture("setlist.hls")).
				Import(context.Background(), tt.file, slot.Address{Slot: 1}, out)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)
				s.Require().NoFileExists(out)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(sdk.Imported, got.Action)
			s.Require().Equal(out, got.Path)
		})
	}
}

// TestCopy covers putting one slot of a file into another.
func (s *SetlistPublicTestSuite) TestCopy() {
	tests := []struct {
		name string
		path string
		says string
	}{
		{name: "one slot onto another", path: fixture("setlist.hls")},
		{name: "in a file that is not there", path: fixture("nope.hls"), says: "opening"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "out.hls")

			got, err := s.setlist(tt.path).
				Copy(context.Background(), slot.Address{}, slot.Address{Slot: 1}, out)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(sdk.Copied, got.Action)
			s.Require().Equal(out, got.Path)
		})
	}
}

// TestSwap covers exchanging two slots of a file.
func (s *SetlistPublicTestSuite) TestSwap() {
	tests := []struct {
		name string
		path string
		says string
	}{
		{name: "two slots", path: fixture("setlist.hls")},
		{name: "in a file that is not there", path: fixture("nope.hls"), says: "opening"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "out.hls")

			got, err := s.setlist(tt.path).
				Swap(context.Background(), slot.Address{}, slot.Address{Slot: 1}, out)

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(sdk.Swapped, got.Action)
			s.Require().Equal(out, got.Path)
		})
	}
}

// orBackground is ctx, or a context nobody has stopped waiting on.
func orBackground(
	ctx context.Context,
) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

func TestSetlistPublicTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(SetlistPublicTestSuite))
}
