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
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// SelectDevicePublicTestSuite covers loading a preset on a device.
type SelectDevicePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	dev  *selectable
}

// selectable is a session that can read and change what is playing.
type selectable struct {
	*mocks.MockEditor
	*mocks.MockSelector
}

func (*selectable) Close() {}

func (s *SelectDevicePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.dev = &selectable{
		MockEditor:   mocks.NewMockEditor(s.ctrl),
		MockSelector: mocks.NewMockSelector(s.ctrl),
	}
}

func (s *SelectDevicePublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// listing is what a device answers when asked what a setlist holds.
func (s *SelectDevicePublicTestSuite) listing() []wire.Preset {
	return []wire.Preset{{Slot: 0, Name: "Chunky Monkey"}, {Slot: 4, Name: "Montana"}}
}

func (s *SelectDevicePublicTestSuite) TestLoadsAPresetAndNamesIt() {
	s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, 4).Return(nil)

	var out bytes.Buffer

	s.Require().NoError(slots.SelectWith(context.Background(), &out, s.dev,
		slots.DeviceOptions{Slot: 4}))

	s.Require().Contains(out.String(), "02B")
	s.Require().Contains(out.String(), "Montana")
	s.Require().Contains(out.String(), "loaded")
}

func (s *SelectDevicePublicTestSuite) TestReportsWhatStopsIt() {
	tests := []struct {
		name   string
		expect func()
		want   string
	}{
		{
			name: "a device that will not say what it holds",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(nil, errors.New("no answer"))
			},
			want: "listing setlist 0",
		},
		{
			name: "a device that will not switch",
			expect: func() {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockSelector.EXPECT().
					SelectPreset(gomock.Any(), 0, 4).
					Return(errors.New("still switching"))
			},
			want: "selecting slot 02B",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.expect()

			err := slots.SelectWith(context.Background(), &bytes.Buffer{}, s.dev,
				slots.DeviceOptions{Slot: 4})

			s.Require().ErrorContains(err, tt.want)
		})
	}
}

// TestNeedsASessionThatCanSelect keeps a reading session from being handed
// the ability to change what somebody is hearing.
func (s *SelectDevicePublicTestSuite) TestNeedsASessionThatCanSelect() {
	err := slots.SelectWith(context.Background(), &bytes.Buffer{},
		mocks.NewMockEditor(s.ctrl), slots.DeviceOptions{Slot: 4})

	s.Require().ErrorContains(err, "cannot select")
}

// TestReportsAFailingWriter covers somewhere for the result to go.
func (s *SelectDevicePublicTestSuite) TestReportsAFailingWriter() {
	s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, 4).Return(nil)

	s.Require().Error(slots.SelectWith(context.Background(), &brokenWriter{},
		s.dev, slots.DeviceOptions{Slot: 4}))
}

// TestSelectFindsItsOwnDevice covers the entry point somebody runs, which is
// one line: find a session, hand it on, release it.
func (s *SelectDevicePublicTestSuite) TestSelectFindsItsOwnDevice() {
	s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
	s.dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, 4).Return(nil)

	defer s.stand(s.dev, nil)()

	var out bytes.Buffer

	s.Require().NoError(slots.SelectDevice(context.Background(), &out,
		slots.DeviceOptions{Slot: 4}))

	s.Require().Contains(out.String(), "loaded")
}

// TestSelectReportsNoDeviceAttached covers finding none.
func (s *SelectDevicePublicTestSuite) TestSelectReportsNoDeviceAttached() {
	defer s.stand(nil, errors.New("nothing on the bus"))()

	s.Require().ErrorContains(slots.SelectDevice(context.Background(),
		&bytes.Buffer{}, slots.DeviceOptions{Slot: 4}), "nothing on the bus")
}

// stand puts a session in place of the one that needs hardware, and takes it
// away again.
func (s *SelectDevicePublicTestSuite) stand(
	dev sdk.Editor,
	err error,
) func() {
	restore := *slots.OpenDevice
	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return dev, err
	}

	return func() { *slots.OpenDevice = restore }
}

func TestSelectDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(SelectDevicePublicTestSuite))
}
