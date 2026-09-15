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
package mcp_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// HeldPublicTestSuite covers telling whether the server holds the pedal.
type HeldPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *HeldPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
}

// TestHeld covers Held while the pedal is claimed, after a device call, and
// once the server stops.
func (s *HeldPublicTestSuite) TestHeld() {
	gone := fmt.Errorf("%w: reading from the device: gone", sdk.ErrBus)
	refused := errors.New("HX Edit has the pedal")

	tests := []struct {
		name string
		// setup says what the client and its Session expect. claiming is
		// called while the pedal is being claimed.
		setup func(c *mocks.MockClient, session *mocks.MockSession, claiming func())
		// call is whether a device tool is called.
		call bool
		// whileClaiming is Held while the pedal is being claimed.
		whileClaiming bool
		// after is Held once the call has answered.
		after bool
	}{
		{
			name:  "a server no device tool has been called on",
			setup: func(*mocks.MockClient, *mocks.MockSession, func()) {},
		},
		{
			// The Session stays open between calls, until the agent has been
			// quiet a while or the server stops.
			name: "a device call that opened a Session",
			setup: func(c *mocks.MockClient, session *mocks.MockSession, claiming func()) {
				c.EXPECT().Open(gomock.Any()).DoAndReturn(
					func(context.Context) (tools.Session, error) {
						claiming()

						return session, nil
					})
				session.EXPECT().Presets(gomock.Any(), 0).Return(sdk.Listing{}, nil)
				session.EXPECT().Close().Return(nil)
			},
			call:          true,
			whileClaiming: true,
			after:         true,
		},
		{
			name: "a claim the pedal refused",
			setup: func(c *mocks.MockClient, _ *mocks.MockSession, claiming func()) {
				c.EXPECT().Open(gomock.Any()).DoAndReturn(
					func(context.Context) (tools.Session, error) {
						claiming()

						return nil, refused
					})
			},
			call:          true,
			whileClaiming: true,
		},
		{
			// A bus error finishes the Session, so it is let go at once.
			name: "a device call the bus failed",
			setup: func(c *mocks.MockClient, session *mocks.MockSession, claiming func()) {
				c.EXPECT().Open(gomock.Any()).DoAndReturn(
					func(context.Context) (tools.Session, error) {
						claiming()

						return session, nil
					})
				session.EXPECT().Presets(gomock.Any(), 0).Return(sdk.Listing{}, gone)
				session.EXPECT().Close().Return(gone)
			},
			call:          true,
			whileClaiming: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := mocks.NewMockClient(s.ctrl)
			session := mocks.NewMockSession(s.ctrl)
			server := mcp.NewOver(c, mcp.Options{})

			var claimed atomic.Bool

			tt.setup(c, session, func() { claimed.Store(server.Held()) })

			serverEnd, clientEnd := gomcp.NewInMemoryTransports()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			served := make(chan error, 1)
			go func() { served <- server.Serve(ctx, serverEnd) }()

			agent, err := gomcp.NewClient(
				&gomcp.Implementation{Name: "test", Version: "test"}, nil,
			).Connect(context.Background(), clientEnd, nil)
			s.Require().NoError(err)

			if tt.call {
				_, err := agent.CallTool(context.Background(), &gomcp.CallToolParams{
					Name:      "presets_list",
					Arguments: map[string]any{},
				})
				s.Require().NoError(err)
			}

			s.Equal(tt.whileClaiming, claimed.Load(), "while the pedal is claimed")
			s.Equal(tt.after, server.Held(), "once the call has answered")

			cancel()
			s.Require().ErrorIs(<-served, context.Canceled)

			s.False(server.Held(), "once the server has stopped")
		})
	}
}

func TestHeldPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeldPublicTestSuite))
}
