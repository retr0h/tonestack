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

// Package mcp serves tonestack's operations to an agent over the Model Context
// Protocol.
//
// It reaches pkg/sdk and nothing else in this module, so it can leave for a
// repository of its own the way pkg/cli can.
package mcp

import (
	"context"
	"errors"
	"io"
	"os"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// instructions are what a connecting agent is told before its first call.
const instructions = `tonestack builds Line 6 Helix presets from rigs, and reads and writes the pedal.

Use catalog_search before naming any model. If it does not find the gear, the
device does not model it: say so, and never invent a model identifier.

Build with preset_build, then read what it added and what each character word
moved before putting the preset on a pedal. Quit HX Edit before any tool that
reaches the pedal.

A preset that builds, a preset HX Edit imports, and a preset the hardware loads
are three different claims. Say which one you have.`

// Options say how the server runs.
type Options struct {
	// Version is what the server reports itself as. Empty reports "dev".
	Version string
	// AllowWrites offers the tools that overwrite what a pedal holds.
	AllowWrites bool
}

// Server is tonestack's MCP server.
type Server struct {
	server *gomcp.Server
	// pedal holds the pedal between device calls.
	pedal io.Closer
}

// New builds a server whose tools call client.
func New(
	client *sdk.Client,
	opts Options,
) *Server {
	version := opts.Version
	if version == "" {
		version = "dev"
	}

	s := gomcp.NewServer(
		&gomcp.Implementation{Name: "tonestack", Version: version},
		&gomcp.ServerOptions{Instructions: instructions},
	)
	pedal := tools.Register(s, tools.FromSDK(client), opts.AllowWrites)

	return &Server{server: s, pedal: pedal}
}

// Run serves over stdin and stdout until ctx ends or the agent disconnects.
//
// An agent disconnecting ends the session cleanly, including when it closes
// stdin while a reply is still being written.
func (s *Server) Run(
	ctx context.Context,
) error {
	return s.run(ctx, os.Stdin, os.Stdout)
}

// Serve serves over any transport until ctx ends or the agent disconnects.
//
// The pedal is let go when the agent is, so its front panel works again and
// HX Edit can claim it.
func (s *Server) Serve(
	ctx context.Context,
	t gomcp.Transport,
) error {
	err := s.server.Run(ctx, t)

	return errors.Join(err, s.pedal.Close())
}
