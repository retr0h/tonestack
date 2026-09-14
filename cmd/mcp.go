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
package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/sdk"
)

var mcpAllowWrites bool

// mcpCmd represents the mcp command.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve tonestack to an agent over MCP",
	Long: `Serve the catalog, the corpus, the shipped rigs, building presets and the
attached pedal to an agent over the Model Context Protocol, on stdin and stdout.

An agent starts this itself. For Claude Code:

  claude mcp add tonestack -- tonestack mcp

Tools that overwrite what a pedal holds (import, copy, swap) are offered only
with --allow-writes. Each still saves what it replaces to a file first.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		err := mcp.New(sdk.New(), mcp.Options{
			Version:     version,
			AllowWrites: mcpAllowWrites,
		}).Run(cmd.Context())

		// Ctrl-C and SIGTERM are how this is meant to stop, not a failure.
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().BoolVar(&mcpAllowWrites, "allow-writes", false,
		"offer the tools that overwrite slots on the pedal")
}
