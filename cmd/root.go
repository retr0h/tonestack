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
// Package cmd holds the command definitions.
//
// This package and main.go are excluded from coverage by .coverignore.
// Behaviour worth testing lives in internal/.
package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/internal/cli"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "tonestack",
	Short: "Describe a guitar or bass sound, get a Line 6 Helix preset",
	Long: `Describe a guitar or bass sound and get a preset file that loads on a
Line 6 Helix device.

Everything needed ships in this binary: the curated gear knowledge, and the
catalog of what the device can do. Nothing else has to be installed, and no
device has to be attached, to describe a chain and write a preset.`,
	Args: cobra.NoArgs,
}

// Execute is called by main.main(). It only needs to happen once to the
// rootCmd.
func Execute() {
	// Shell completion is cobra's, not ours, and listing it beside the
	// commands this tool is for buries them.
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	styleHelp(rootCmd)

	// Cobra prints what went wrong and the flags for the command that would
	// not run. The mark this tool puts in front of a failure goes with it, so
	// what cobra prints reads like everything else this tool says.
	rootCmd.SetErrPrefix(cli.FailurePrefix(os.Stderr))

	// A device command holds a USB interface and gives it back on the way
	// out. Ctrl-C has to reach it: without a context to cancel, the process
	// dies where it stands, the interface is released by teardown rather
	// than by the session that claimed it, and the pedal is left needing a
	// power cycle. See docs/protocol.md.
	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
