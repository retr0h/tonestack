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

package tools_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type DevicePublicTestSuite struct {
	suite.Suite
	client *mocks.MockClient
}

func (s *DevicePublicTestSuite) SetupSubTest() {
	s.client = mocks.NewMockClient(gomock.NewController(s.T()))
}

// deviceRow is one call and what it should come back with. check reads the
// structured answer of a call that succeeded.
type deviceRow struct {
	name  string
	args  any
	setup func(c *mocks.MockClient)
	want  string
	err   bool
	check func(s *DevicePublicTestSuite, res *gomcp.CallToolResult)
	// allowWrites starts the server the way --allow-writes does.
	allowWrites bool
}

func (s *DevicePublicTestSuite) run(
	tool string,
	tests []deviceRow,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup(s.client)
			}

			res := call(s.T(), connect(s.T(), s.client, tt.allowWrites), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if tt.check != nil {
				tt.check(s, res)
			}
		})
	}
}

var errHXEdit = errors.New("the editor interface is in use, quit HX Edit")

// TestDevicesList covers what is attached, and one call at a time.
func (s *DevicePublicTestSuite) TestDevicesList() {
	s.run("devices_list", []deviceRow{
		{
			name: "a pedal attached",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Devices(gomock.Any()).
					Return(sdk.Attached{Devices: []sdk.Attachment{{Model: "HX Stomp"}}}, nil)
			},
			want: "1 attached",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Attached
				structured(s.T(), res, &got)
				s.Equal("HX Stomp", got.Devices[0].Model)
			},
		},
		{
			name: "a bus that will not answer",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Devices(gomock.Any()).
					Return(sdk.Attached{}, errors.New("bus unavailable"))
			},
			want: "bus unavailable",
			err:  true,
		},
	})

	s.Run("two calls at once", func() {
		// A barrier rather than a sleep: the first call holds the device
		// until the test lets it go, so the second has every chance to get
		// in and must not.
		entered := make(chan struct{}, 2)
		release := make(chan struct{})

		s.client.EXPECT().Devices(gomock.Any()).Times(2).DoAndReturn(
			func(context.Context) (sdk.Attached, error) {
				entered <- struct{}{}
				<-release

				return sdk.Attached{}, nil
			})

		session := connect(s.T(), s.client, false)
		devices := func() {
			_, _ = session.CallTool(context.Background(), &gomcp.CallToolParams{
				Name: "devices_list", Arguments: tools.None{},
			})
		}

		var wg sync.WaitGroup
		defer wg.Wait()

		wg.Go(devices)

		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			close(release)
			s.FailNow("the first call never reached the device")
		}

		wg.Go(devices)

		select {
		case <-entered:
			close(release)
			s.FailNow("the second call reached the device while the first held it")
		case <-time.After(200 * time.Millisecond):
		}

		close(release)
		wg.Wait()

		s.Len(entered, 1, "the second call reached the device once the first let go")
	})
}

// TestPresetsList covers reading the setlist.
func (s *DevicePublicTestSuite) TestPresetsList() {
	s.run("presets_list", []deviceRow{
		{
			name: "a setlist",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Presets(gomock.Any(), sdk.Where{}).
					Return(sdk.Listing{Slots: []sdk.Held{{}, {}}}, nil)
			},
			want: "of 2 slots",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Listing
				structured(s.T(), res, &got)
				s.Len(got.Slots, 2)
			},
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Presets(gomock.Any(), sdk.Where{}).Return(sdk.Listing{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetShow covers reading one slot.
func (s *DevicePublicTestSuite) TestPresetShow() {
	s.run("preset_show", []deviceRow{
		{
			name: "a slot by its label",
			args: tools.Slot{Slot: "01B"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Preset(gomock.Any(), sdk.Read{Slot: 1}).
					Return(sdk.Reading{Name: "Chunky Monkey"}, nil)
			},
			want: "01B holds Chunky Monkey",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Shown
				structured(s.T(), res, &got)
				s.Equal("Chunky Monkey", got.Name)
			},
		},
		{
			// "99Z" fails to parse: no bank has a letter past C, so
			// slot.Value.Set refuses it before the client is ever called.
			name: "a label the pedal does not have",
			args: tools.Slot{Slot: "99Z"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Slot{Slot: "01A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Preset(gomock.Any(), sdk.Read{Slot: 0}).Return(sdk.Reading{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetExport covers writing a slot out.
func (s *DevicePublicTestSuite) TestPresetExport() {
	dir := s.T().TempDir()
	fresh := filepath.Join(dir, "fresh.yaml")
	held := filepath.Join(dir, "held.yaml")
	s.Require().NoError(os.WriteFile(held, []byte("somebody's rig"), 0o600))

	s.run("preset_export", []deviceRow{
		{
			name: "a path nothing is at",
			args: tools.Export{Slot: "01A", Out: fresh},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Export(gomock.Any(), sdk.Export{Slot: 0, OutputPath: fresh}).
					Return(sdk.Written{Path: fresh}, nil)
			},
			want: "wrote " + fresh,
		},
		{
			// No client call is expected, so reaching the device fails the
			// row.
			name: "a path a file is at, with writes off",
			args: tools.Export{Slot: "01A", Out: held},
			want: tools.ErrWouldOverwrite.Error() + ": " + held,
			err:  true,
		},
		{
			name: "a path a file is at, with writes on",
			args: tools.Export{Slot: "01A", Out: held},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Export(gomock.Any(), sdk.Export{Slot: 0, OutputPath: held}).
					Return(sdk.Written{Path: held}, nil)
			},
			want:        "wrote " + held,
			allowWrites: true,
		},
		{
			name: "a slot as a rig",
			args: tools.Export{Slot: "01A", Out: "a.yaml"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Export(gomock.Any(), sdk.Export{Slot: 0, OutputPath: "a.yaml"}).
					Return(sdk.Written{Path: "a.yaml"}, nil)
			},
			want: "wrote a.yaml from 01A",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Written
				structured(s.T(), res, &got)
				s.Equal("a.yaml", got.Path)
			},
		},
		{
			// "nope" fails to parse: its trailing letter, E, is past the
			// last bank letter C, so slot.Value.Set refuses it before the
			// client is ever called.
			name: "a label the pedal does not have",
			args: tools.Export{Slot: "nope", Out: "a.yaml"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Export{Slot: "01A", Out: "a.hlx", As: "hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Export(gomock.Any(), sdk.Export{Slot: 0, OutputPath: "a.hlx", As: "hlx"}).
					Return(sdk.Written{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetSelect covers loading a slot.
func (s *DevicePublicTestSuite) TestPresetSelect() {
	s.run("preset_select", []deviceRow{
		{
			name: "a slot by its label",
			args: tools.Slot{Slot: "07A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Select(gomock.Any(), sdk.Read{Slot: 18}).
					Return(sdk.Change{Action: sdk.Selected, To: sdk.At{Slot: 18, Name: "Chunky Monkey"}}, nil)
			},
			want: "loaded 07A",
			check: func(s *DevicePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Change
				structured(s.T(), res, &got)
				s.Equal(sdk.Selected, got.Action)
			},
		},
		{
			// "0A" fails to parse: there is no bank zero, banks count from
			// one, so slot.Value.Set refuses it before the client is ever
			// called. ("43A" was tried first, but a bank number carries no
			// upper bound in slot.parse, so it decodes to a slot index
			// rather than failing.)
			name: "a label the pedal does not have",
			args: tools.Slot{Slot: "0A"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Slot{Slot: "07A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Select(gomock.Any(), gomock.Any()).Return(sdk.Change{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
