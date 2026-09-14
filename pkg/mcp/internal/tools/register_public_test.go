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
	"encoding/json"
	"slices"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
)

// connect puts the tools on a server and a client session in front of it,
// over the library's in-memory transport, so a test calls a tool the way an
// agent does.
func connect(
	t *testing.T,
	c tools.Client,
	allowWrites bool,
) *gomcp.ClientSession {
	t.Helper()

	server := gomcp.NewServer(&gomcp.Implementation{Name: "tonestack", Version: "test"}, nil)
	tools.Register(server, c, allowWrites)

	serverEnd, clientEnd := gomcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, serverEnd, nil)
	require.NoError(t, err)

	session, err := gomcp.NewClient(
		&gomcp.Implementation{Name: "test", Version: "test"}, nil,
	).Connect(ctx, clientEnd, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// call calls one tool and fails the test only on a protocol error. A tool
// error comes back as a result with IsError set, which is what the tests read.
func call(
	t *testing.T,
	session *gomcp.ClientSession,
	name string,
	args any,
) *gomcp.CallToolResult {
	t.Helper()

	res, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)

	return res
}

// text is the line a tool said.
func text(
	t *testing.T,
	res *gomcp.CallToolResult,
) string {
	t.Helper()
	require.NotEmpty(t, res.Content)

	said, ok := res.Content[0].(*gomcp.TextContent)
	require.True(t, ok, "first content is %T", res.Content[0])

	return said.Text
}

// structured decodes what a tool answered into a Go value.
func structured(
	t *testing.T,
	res *gomcp.CallToolResult,
	into any,
) {
	t.Helper()

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, into))
}

type RegisterPublicTestSuite struct {
	suite.Suite
}

// TestRegister covers which tools an agent is offered.
func (s *RegisterPublicTestSuite) TestRegister() {
	reads := []string{
		"catalog_block", "catalog_search", "corpus_model",
		"preset_build", "rig_show", "rigs_list",
		"devices_list", "presets_list", "preset_show", "preset_export", "preset_select",
	}

	tests := []struct {
		name        string
		allowWrites bool
		want        []string
		readOnly    map[string]bool
	}{
		{
			name: "without writes",
			want: reads,
			readOnly: map[string]bool{
				"catalog_block": true, "catalog_search": true, "corpus_model": true,
				"preset_build": false, "rig_show": true, "rigs_list": true,
				"devices_list": true, "presets_list": true, "preset_show": true,
				"preset_export": true, "preset_select": false,
			},
		},
		{
			name:        "with writes",
			allowWrites: true,
			want: append(
				slices.Clone(reads),
				"preset_import",
				"presets_copy",
				"presets_swap",
			),
			readOnly: map[string]bool{
				"preset_import": false,
				"presets_copy":  false,
				"presets_swap":  false,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			client := mocks.NewMockClient(gomock.NewController(s.T()))
			session := connect(s.T(), client, tt.allowWrites)

			listed, err := session.ListTools(context.Background(), nil)
			s.Require().NoError(err)

			var names []string
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)

				if want, ok := tt.readOnly[tool.Name]; ok {
					s.Equal(want, tool.Annotations.ReadOnlyHint, tool.Name)
				}

				if tool.Name == "preset_select" {
					s.Require().NotNil(tool.Annotations.DestructiveHint)
					s.False(*tool.Annotations.DestructiveHint)
					s.True(tool.Annotations.IdempotentHint)
				}

				switch tool.Name {
				case "preset_import", "presets_copy", "presets_swap":
					s.Require().NotNil(tool.Annotations.DestructiveHint, tool.Name)
					s.True(*tool.Annotations.DestructiveHint, tool.Name)
				}
			}

			s.ElementsMatch(tt.want, names)
		})
	}
}

func TestRegisterPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterPublicTestSuite))
}
