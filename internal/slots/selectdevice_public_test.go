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
	"io"
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

// TestSelectWith loads a preset on a session.
func (s *SelectDevicePublicTestSuite) TestSelectWith() {
	tests := []struct {
		name string
		// whether the listing comes back, and whether the device switches. An
		// empty outcome means the call is never reached.
		listed   bool
		selects  string
		readOnly bool
		deaf     bool

		contains []string
		errText  string
	}{
		{
			name:     "a preset the device holds",
			listed:   true,
			selects:  "loaded",
			contains: []string{"02B", "Montana", "loaded"},
		},
		{
			name:    "a device that will not say what it holds",
			errText: "listing setlist 0",
		},
		{
			name:    "a device that will not switch",
			listed:  true,
			selects: "refused",
			errText: "selecting slot 02B",
		},
		{
			// A reading session must not be handed the ability to change what
			// somebody is hearing.
			name:     "a session that cannot select",
			readOnly: true,
			errText:  "cannot select",
		},
		{
			name:    "a writer with nowhere for the result to go",
			listed:  true,
			selects: "loaded",
			deaf:    true,
			errText: "boom",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dev := sdk.Editor(s.dev)

			if tt.readOnly {
				dev = mocks.NewMockEditor(s.ctrl)
			} else if tt.listed {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
			} else {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(nil, errors.New("no answer"))
			}

			switch tt.selects {
			case "loaded":
				s.dev.MockSelector.EXPECT().
					SelectPreset(gomock.Any(), 0, 4).Return(nil)
			case "refused":
				s.dev.MockSelector.EXPECT().
					SelectPreset(gomock.Any(), 0, 4).
					Return(errors.New("still switching"))
			}

			var out bytes.Buffer

			w := io.Writer(&out)
			if tt.deaf {
				w = &brokenWriter{}
			}

			err := slots.SelectWith(
				context.Background(), w, dev, slots.DeviceOptions{Slot: 4})

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}
		})
	}
}

// TestSelectDevice covers the entry point somebody runs, which is one line:
// find a session, hand it on, release it.
func (s *SelectDevicePublicTestSuite) TestSelectDevice() {
	tests := []struct {
		name     string
		attached bool
		contains string
		errText  string
	}{
		{name: "a device on the bus", attached: true, contains: "loaded"},
		{name: "nothing on the bus", errText: "nothing on the bus"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.attached {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(s.listing(), nil)
				s.dev.MockSelector.EXPECT().
					SelectPreset(gomock.Any(), 0, 4).Return(nil)

				defer s.stand(s.dev, nil)()
			} else {
				defer s.stand(nil, errors.New("nothing on the bus"))()
			}

			var out bytes.Buffer

			err := slots.SelectDevice(
				context.Background(), &out, slots.DeviceOptions{Slot: 4})

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(out.String(), tt.contains)
		})
	}
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
