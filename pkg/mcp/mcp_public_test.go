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
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
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

			dir := s.T().TempDir()
			fromRecipe := filepath.Join(dir, "recipe.hlx")
			fromRig := filepath.Join(dir, "rig.hlx")

			// Offline tools only, against the real catalog, corpus and rigs: the
			// SDK validates each answer against its declared schema, and only
			// real data shows whether the two agree.
			calls := []struct {
				tool  string
				args  map[string]string
				check func(res *gomcp.CallToolResult)
			}{
				{
					tool: "catalog_search",
					args: map[string]string{"search": "SVT"},
					check: func(res *gomcp.CallToolResult) {
						var got sdk.Blocks
						s.decode(res, &got)
						s.NotEmpty(got.Matched)
					},
				},
				{
					tool: "rigs_list",
					args: map[string]string{},
					check: func(res *gomcp.CallToolResult) {
						var got sdk.Recipes
						s.decode(res, &got)
						s.NotEmpty(got.Rigs)
					},
				},
				{
					tool: "rig_show",
					args: map[string]string{"id": "mike-dirnt"},
					check: func(res *gomcp.CallToolResult) {
						var got sdk.Recipe
						s.decode(res, &got)
						s.Equal("mike-dirnt", got.Rig.ID)
					},
				},
				{
					tool: "corpus_model",
					args: map[string]string{"id": "HD2_AmpSVBeastBrt"},
					check: func(res *gomcp.CallToolResult) {
						var got tools.Model
						s.decode(res, &got)
						s.NotZero(got.Uses)
					},
				},
				{
					tool: "preset_build",
					args: map[string]string{"recipe_id": "mike-dirnt", "out": fromRecipe},
					check: func(res *gomcp.CallToolResult) {
						var got built
						s.decode(res, &got)
						s.Require().NotNil(got.FromRecipe)
						s.Equal(fromRecipe, got.FromRecipe.Path)
					},
				},
				{
					tool: "preset_build",
					args: map[string]string{
						"rig_path": filepath.Join(
							"..",
							"..",
							"examples",
							"rigspec",
							"mike-dirnt.yaml",
						),
						"out": fromRig,
					},
					check: func(res *gomcp.CallToolResult) {
						var got built
						s.decode(res, &got)
						s.Require().NotNil(got.FromRig)
						s.Equal(fromRig, got.FromRig.Path)
					},
				},
			}

			for _, c := range calls {
				res, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
					Name:      c.tool,
					Arguments: c.args,
				})
				s.Require().NoError(err, c.tool)
				s.Require().False(res.IsError, "%s: %v", c.tool, res.Content)
				c.check(res)
			}

			cancel()
			s.ErrorIs(<-served, context.Canceled)
		})
	}
}

// built is preset_build's answer as an agent reads it.
type built struct {
	FromRecipe *sdk.Made  `json:"from_recipe"`
	FromRig    *sdk.Built `json:"from_rig"`
}

// decode reads a tool's structured answer into a Go value.
func (s *MCPPublicTestSuite) decode(
	res *gomcp.CallToolResult,
	into any,
) {
	raw, err := json.Marshal(res.StructuredContent)
	s.Require().NoError(err)
	s.Require().NoError(json.Unmarshal(raw, into))
}

// TestRun covers stdio, which ends when the command's context does.
func (s *MCPPublicTestSuite) TestRun() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- mcp.New(sdk.New(), mcp.Options{}).Run(ctx)
	}()

	select {
	case err := <-done:
		s.ErrorIs(err, context.Canceled)
	case <-time.After(2 * time.Second):
		s.Fail("Run did not return after its context ended")
	}
}

func TestMCPPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MCPPublicTestSuite))
}
