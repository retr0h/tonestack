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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// PedalPublicTestSuite covers holding one Session across the device tools.
type PedalPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *PedalPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
}

// step is one tool call, and whether it should fail.
type step struct {
	tool string
	args any
	err  bool
	// afterIdle waits for the first Session to be let go before calling.
	afterIdle bool
}

// TestOnPedal covers what a device tool does with the pedal.
func (s *PedalPublicTestSuite) TestOnPedal() {
	listing := sdk.Listing{Slots: []sdk.Held{{}}}
	gone := fmt.Errorf("%w: reading from the device: gone", sdk.ErrBus)
	list := step{tool: "presets_list", args: tools.None{}}
	listFails := step{tool: "presets_list", args: tools.None{}, err: true}

	tests := []struct {
		name string
		idle time.Duration
		// setup says what the client and the two Sessions it may open
		// expect. letGo is called when the first Session is closed.
		setup func(c *mocks.MockClient, first, second *mocks.MockSession, letGo func())
		steps []step
	}{
		{
			// An agent's list, show and select cost one handshake.
			name: "three device tools on one open",
			setup: func(c *mocks.MockClient, first, _ *mocks.MockSession, _ func()) {
				c.EXPECT().Open(gomock.Any()).Return(first, nil)
				first.EXPECT().Presets(gomock.Any(), 0).Return(listing, nil)
				first.EXPECT().Preset(gomock.Any(), slot.Address{}).
					Return(sdk.Reading{Name: "Longview"}, nil)
				first.EXPECT().Select(gomock.Any(), slot.Address{}).
					Return(sdk.Change{Action: sdk.Selected}, nil)
				first.EXPECT().Close().Return(nil)
			},
			steps: []step{
				list,
				{tool: "preset_show", args: tools.Slot{Slot: "01A"}},
				{tool: "preset_select", args: tools.Slot{Slot: "01A"}},
			},
		},
		{
			// The front panel stops refreshing footswitches while an editor
			// is attached, so a quiet pedal is let go.
			name: "a pedal nobody has called for a while",
			idle: 20 * time.Millisecond,
			setup: func(c *mocks.MockClient, first, second *mocks.MockSession, letGo func()) {
				gomock.InOrder(
					c.EXPECT().Open(gomock.Any()).Return(first, nil),
					c.EXPECT().Open(gomock.Any()).Return(second, nil),
				)
				first.EXPECT().Presets(gomock.Any(), 0).Return(listing, nil)
				first.EXPECT().Close().DoAndReturn(func() error {
					letGo()

					return nil
				})
				second.EXPECT().Presets(gomock.Any(), 0).Return(listing, nil)
				second.EXPECT().Close().Return(nil).MaxTimes(1)
			},
			steps: []step{list, {tool: "presets_list", args: tools.None{}, afterIdle: true}},
		},
		{
			// The Session is finished, so it is let go, and the error is
			// what the agent sees. Calling again is the agent's choice, and
			// that call opens a fresh Session: nothing here reconnects.
			name: "a call the bus fails",
			setup: func(c *mocks.MockClient, first, second *mocks.MockSession, _ func()) {
				gomock.InOrder(
					c.EXPECT().Open(gomock.Any()).Return(first, nil),
					c.EXPECT().Open(gomock.Any()).Return(second, nil),
				)
				first.EXPECT().Presets(gomock.Any(), 0).Return(sdk.Listing{}, gone)
				first.EXPECT().Close().Return(gone)
				second.EXPECT().Presets(gomock.Any(), 0).Return(listing, nil)
				second.EXPECT().Close().Return(nil)
			},
			steps: []step{listFails, list},
		},
		{
			// A device that did not answer in time is still there, and the
			// Session with it is still good.
			name: "a call that fails some other way",
			setup: func(c *mocks.MockClient, first, _ *mocks.MockSession, _ func()) {
				c.EXPECT().Open(gomock.Any()).Return(first, nil)
				gomock.InOrder(
					first.EXPECT().Presets(gomock.Any(), 0).
						Return(sdk.Listing{}, errors.New("no reply")),
					first.EXPECT().Presets(gomock.Any(), 0).Return(listing, nil),
				)
				first.EXPECT().Close().Return(nil)
			},
			steps: []step{listFails, list},
		},
		{
			name: "a pedal that will not open",
			setup: func(c *mocks.MockClient, first, _ *mocks.MockSession, _ func()) {
				gomock.InOrder(
					c.EXPECT().Open(gomock.Any()).Return(nil, errHXEdit),
					c.EXPECT().Open(gomock.Any()).Return(first, nil),
				)
				first.EXPECT().Presets(gomock.Any(), 0).Return(listing, nil)
				first.EXPECT().Close().Return(nil)
			},
			steps: []step{listFails, list},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := mocks.NewMockClient(s.ctrl)
			first, second := mocks.NewMockSession(s.ctrl), mocks.NewMockSession(s.ctrl)

			var once sync.Once

			idled := make(chan struct{})
			tt.setup(c, first, second, func() { once.Do(func() { close(idled) }) })

			idle := tt.idle
			if idle == 0 {
				idle = time.Hour
			}

			session, _ := connectIdle(s.T(), c, false, idle)

			for i, st := range tt.steps {
				if st.afterIdle {
					select {
					case <-idled:
					case <-time.After(5 * time.Second):
						s.FailNow("the idle pedal was never let go")
					}
				}

				res := call(s.T(), session, st.tool, st.args)
				s.Equal(st.err, res.IsError, "call %d: %v", i, res.Content)
			}
		})
	}
}

// TestOnPedalPanicking covers a call that panics, which cannot go through a
// server: go-sdk does not recover the handler, so the process would die.
func (s *PedalPublicTestSuite) TestOnPedalPanicking() {
	s.Run("a call that panics lets the Session go", func() {
		c := mocks.NewMockClient(s.ctrl)
		first, second := mocks.NewMockSession(s.ctrl), mocks.NewMockSession(s.ctrl)

		gomock.InOrder(
			c.EXPECT().Open(gomock.Any()).Return(first, nil),
			c.EXPECT().Open(gomock.Any()).Return(second, nil),
		)
		first.EXPECT().Close().Return(nil)
		second.EXPECT().Close().Return(nil)

		p := tools.NewPedal(c)

		s.Require().PanicsWithValue("a bug inside a tool", func() {
			_, _ = tools.OnPedal(context.Background(), p, func(tools.Session) (int, error) {
				panic("a bug inside a tool")
			})
		})

		// Let go on the way up, so the next call opens a fresh Session.
		got, err := tools.OnPedal(context.Background(), p, func(tools.Session) (int, error) {
			return 1, nil
		})
		s.Require().NoError(err)
		s.Require().Equal(1, got)
		s.Require().NoError(p.Close())
	})
}

// TestClose covers letting the pedal go when the server stops.
func (s *PedalPublicTestSuite) TestClose() {
	ended := errors.New("the read loop ended")

	tests := []struct {
		name string
		open bool
		err  error
	}{
		{name: "a pedal holding a Session", open: true, err: ended},
		{name: "one holding nothing"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := mocks.NewMockClient(s.ctrl)

			if tt.open {
				first := mocks.NewMockSession(s.ctrl)
				c.EXPECT().Open(gomock.Any()).Return(first, nil)
				first.EXPECT().Presets(gomock.Any(), 0).Return(sdk.Listing{}, nil)
				first.EXPECT().Close().Return(tt.err)
			}

			session, pedal := connectIdle(s.T(), c, false, time.Hour)

			if tt.open {
				s.False(call(s.T(), session, "presets_list", tools.None{}).IsError)
			}

			if tt.err != nil {
				s.Require().ErrorIs(pedal.Close(), tt.err)
			} else {
				s.Require().NoError(pedal.Close())
			}

			s.Require().NoError(pedal.Close(), "nothing is held the second time")

			// A call that arrives after the server let go does not claim the
			// pedal again.
			res := call(s.T(), session, "presets_list", tools.None{})
			s.True(res.IsError)
			s.Contains(text(s.T(), res), "the server has stopped")
		})
	}
}

func TestPedalPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PedalPublicTestSuite))
}
