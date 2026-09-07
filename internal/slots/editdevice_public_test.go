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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
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
func (s *EditDevicePublicTestSuite) answer() string {
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "pkg", "sdk", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return string(raw)
}

// listing is what the device says the setlist holds.
func (s *EditDevicePublicTestSuite) listing() []wire.Preset {
	return []wire.Preset{
		{Slot: 0, Name: "Chunky Monkey"},
		{Slot: 3, Name: "Black Rusty"},
	}
}

func (s *EditDevicePublicTestSuite) TestCopyingSendsBackWhatItRead() {
	// Byte for byte. A preset that changed on the way through would leave the
	// device's offset table pointing at the wrong places, and the device
	// would accept it and then read the preset as empty.
	body := s.answer()

	s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(body, nil)
	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, 3, "Chunky Monkey", []byte(body)).
		Return(nil)

	var out bytes.Buffer
	s.Require().NoError(slots.CopyWith(context.Background(), &out, s.dev,
		slots.EditOptions{FromSlot: 0, ToSlot: 3}))

	got := out.String()
	s.Require().Contains(got, "01A")
	s.Require().Contains(got, "02A")
	s.Require().Contains(got, "Chunky Monkey")
	s.Require().Contains(got, "replacing Black Rusty",
		"the destination is overwritten and there is no undo on a device")
}

func (s *EditDevicePublicTestSuite) TestSwappingReadsBothBeforeWritingEither() {
	// A device that failed halfway through would otherwise leave one slot
	// holding a copy of the other and the original gone.
	body := s.answer()

	s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)

	first := s.dev.MockEditor.EXPECT().
		ReadPreset(gomock.Any(), 0, 0).Return(body, nil)
	second := s.dev.MockEditor.EXPECT().
		ReadPreset(gomock.Any(), 0, 3).Return(body, nil).After(first)

	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, 3, "Chunky Monkey", []byte(body)).
		Return(nil).After(second)
	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, 0, "Black Rusty", []byte(body)).
		Return(nil).After(second)

	var out bytes.Buffer
	s.Require().NoError(slots.SwapWith(context.Background(), &out, s.dev,
		slots.EditOptions{FromSlot: 0, ToSlot: 3}))

	s.Require().Contains(out.String(), "swapped")
}

func (s *EditDevicePublicTestSuite) TestReportsWhatItCannotDo() {
	body := s.answer()

	tests := []struct {
		name    string
		expect  func()
		message string
	}{
		{
			name: "a listing it cannot get",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(nil, errors.New("boom"))
			},
			message: "listing presets",
		},
		{
			name: "a slot it cannot read",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(nil, errors.New("boom"))
			},
			message: "reading slot 01A",
		},
		{
			name: "an answer that is not a preset",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(map[any]any{}, nil)
			},
			message: "did not answer with a preset",
		},
		{
			name: "a slot it cannot write",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(body, nil)
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 3, gomock.Any(), gomock.Any()).
					Return(errors.New("boom"))
			},
			message: "writing slot 02A",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.expect()

			err := slots.CopyWith(context.Background(), &bytes.Buffer{}, s.dev,
				slots.EditOptions{FromSlot: 0, ToSlot: 3})

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *EditDevicePublicTestSuite) TestSwappingReportsWhatItCannotDo() {
	body := s.answer()

	tests := []struct {
		name    string
		expect  func()
		message string
	}{
		{
			name: "the second slot it cannot read",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(body, nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 3).
					Return(nil, errors.New("boom"))
			},
			message: "reading slot 02A",
		},
		{
			name: "a listing it cannot get",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(nil, errors.New("boom"))
			},
			message: "listing presets",
		},
		{
			name: "the first slot it cannot read",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(nil, errors.New("boom"))
			},
			message: "reading slot 01A",
		},
		{
			name: "the destination it cannot write",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(body, nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 3).
					Return(body, nil)
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 3, gomock.Any(), gomock.Any()).
					Return(errors.New("boom"))
			},
			message: "writing slot 02A",
		},
		{
			name: "the first slot it cannot write",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).
					Return(body, nil)
				s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 3).
					Return(body, nil)
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 3, gomock.Any(), gomock.Any()).
					Return(nil)
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 0, gomock.Any(), gomock.Any()).
					Return(errors.New("boom"))
			},
			message: "writing slot 01A",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.expect()

			err := slots.SwapWith(context.Background(), &bytes.Buffer{}, s.dev,
				slots.EditOptions{FromSlot: 0, ToSlot: 3})

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *EditDevicePublicTestSuite) TestASwapNeedsASessionThatCanWrite() {
	readOnly := mocks.NewMockEditor(s.ctrl)
	readOnly.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	readOnly.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer(), nil)
	readOnly.EXPECT().ReadPreset(gomock.Any(), 0, 3).Return(s.answer(), nil)

	err := slots.SwapWith(context.Background(), &bytes.Buffer{},
		sdk.Editor(readOnly), slots.EditOptions{FromSlot: 0, ToSlot: 3})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot write")
}

func (s *EditDevicePublicTestSuite) TestTheCommandsThatFindTheirOwnDevice() {
	// One line each: find a session, hand it on, release it.
	restore := *slots.OpenDevice
	defer func() { *slots.OpenDevice = restore }()

	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return s.dev, nil
	}

	body := s.answer()

	s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
		Return(s.listing(), nil).Times(2)
	s.dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).
		Return(body, nil).Times(3)
	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).Times(3)

	ctx := context.Background()
	opts := slots.EditOptions{FromSlot: 0, ToSlot: 3}

	s.Require().NoError(slots.CopyDevice(ctx, &bytes.Buffer{}, opts))
	s.Require().NoError(slots.SwapDevice(ctx, &bytes.Buffer{}, opts))
}

func (s *EditDevicePublicTestSuite) TestReportsADeviceItCannotOpen() {
	restore := *slots.OpenDevice
	defer func() { *slots.OpenDevice = restore }()

	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return nil, errors.New("no device found")
	}

	ctx := context.Background()
	opts := slots.EditOptions{FromSlot: 0, ToSlot: 3}

	s.Require().Error(slots.CopyDevice(ctx, &bytes.Buffer{}, opts))
	s.Require().Error(slots.SwapDevice(ctx, &bytes.Buffer{}, opts))
}

func (s *EditDevicePublicTestSuite) TestASessionThatCannotWrite() {
	// Reading and writing are separate abilities, because writing is the half
	// that can destroy somebody's work.
	readOnly := mocks.NewMockEditor(s.ctrl)
	readOnly.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	readOnly.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer(), nil)

	err := slots.CopyWith(context.Background(), &bytes.Buffer{},
		sdk.Editor(readOnly), slots.EditOptions{FromSlot: 0, ToSlot: 3})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot write")
}

func TestEditDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(EditDevicePublicTestSuite))
}
