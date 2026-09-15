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

package cmd_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/cmd"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// MCPStartPublicTestSuite covers the MCP server as the command starts it.
type MCPStartPublicTestSuite struct {
	suite.Suite
}

// reply is one line the server writes, as far as these tests read it.
type reply struct {
	ID     int `json:"id"`
	Result struct {
		StructuredContent sdk.Recipes `json:"structuredContent"`
	} `json:"result"`
	Error json.RawMessage `json:"error"`
}

// TestUserRecipes covers an agent reaching somebody's own recipes through the
// server `tonestack mcp start` runs.
func (s *MCPStartPublicTestSuite) TestUserRecipes() {
	data := s.T().TempDir()
	s.T().Setenv("XDG_DATA_HOME", data)

	artists := filepath.Join(data, "tonestack", "recipes", "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))
	s.Require().NoError(os.WriteFile(filepath.Join(artists, "their-player.yaml"),
		[]byte(theirs("their-player", "")), 0o600))

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	var errs bytes.Buffer

	root := cmd.Root()
	reset(root)
	root.SetIn(inR)
	root.SetOut(outW)
	root.SetErr(&errs)
	root.SetArgs([]string{"mcp", "start"})

	done := make(chan error, 1)
	go func() {
		done <- root.ExecuteContext(context.Background())
		_ = outW.Close()
	}()

	// Each write in its own goroutine: the server may be writing a reply
	// nobody has read yet while a request waits to be read.
	send := func(line string) {
		go func() { _, _ = io.WriteString(inW, line+"\n") }()
	}

	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
		`{"protocolVersion":"2025-06-18","capabilities":{},` +
		`"clientInfo":{"name":"test","version":"0"}}}`)

	lines := bufio.NewScanner(outR)
	lines.Buffer(make([]byte, 0, 1<<20), 64<<20)

	var listed []string

	for lines.Scan() {
		var got reply
		s.Require().NoError(json.Unmarshal(lines.Bytes(), &got), lines.Text())
		s.Require().Empty(got.Error, lines.Text())

		if got.ID == 1 {
			send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
			send(`{"jsonrpc":"2.0","id":2,"method":"tools/call",` +
				`"params":{"name":"rigs_list","arguments":{}}}`)

			continue
		}

		if got.ID == 2 {
			for _, r := range got.Result.StructuredContent.Rigs {
				listed = append(listed, r.ID)
			}

			break
		}
	}

	// Hanging up ends the session, the way an agent closing stdin does.
	_ = inW.Close()

	go func() { _, _ = io.Copy(io.Discard, outR) }()

	select {
	case err := <-done:
		s.Require().NoError(err, errs.String())
	case <-time.After(10 * time.Second):
		s.Fail("mcp start did not end when its input did")
	}

	s.Require().Contains(listed, "their-player", "the agent sees their recipe")
	s.Require().Contains(listed, "mike-dirnt", "beside the ones that ship")
}

func TestMCPStartPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MCPStartPublicTestSuite))
}
