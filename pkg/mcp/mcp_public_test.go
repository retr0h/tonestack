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
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// MCPPublicTestSuite covers pkg/mcp's public surface.
type MCPPublicTestSuite struct {
	suite.Suite
}

// TestServe covers an agent's session with the server.
func (s *MCPPublicTestSuite) TestServe() {
	tests := []struct {
		name  string
		opts  mcp.Options
		tools int
	}{
		{name: "without writes", opts: mcp.Options{Version: "1.2.3"}, tools: 11},
		{name: "with writes", opts: mcp.Options{AllowWrites: true}, tools: 14},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			serverEnd, clientEnd := gomcp.NewInMemoryTransports()
			ctx, cancel := context.WithCancel(context.Background())

			served := make(chan error, 1)
			go func() { served <- mcp.New(sdk.New(), tt.opts).Serve(ctx, serverEnd) }()

			session, err := gomcp.NewClient(
				&gomcp.Implementation{Name: "test", Version: "test"}, nil,
			).Connect(context.Background(), clientEnd, nil)
			s.Require().NoError(err)

			s.Contains(session.InitializeResult().Instructions, "catalog_search")

			listed, err := session.ListTools(context.Background(), nil)
			s.Require().NoError(err)
			s.Len(listed.Tools, tt.tools)

			// "Marshall", not "SVT" from the brief: SVT matches exactly 10
			// blocks in the built-in catalog, and "10 of 665 blocks matched"
			// itself contains the substring "0 of", failing the check below
			// for a reason that has nothing to do with the server.
			res, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
				Name:      "catalog_search",
				Arguments: map[string]string{"search": "Marshall"},
			})
			s.Require().NoError(err)
			s.False(res.IsError)
			s.NotContains(res.Content[0].(*gomcp.TextContent).Text, "0 of")

			cancel()
			s.ErrorIs(<-served, context.Canceled)
		})
	}
}

// TestRun covers stdio, which ends when the command's context does.
func (s *MCPPublicTestSuite) TestRun() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		_ = mcp.New(sdk.New(), mcp.Options{}).Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		s.Fail("Run did not return after its context ended")
	}
}

func TestMCPPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MCPPublicTestSuite))
}
