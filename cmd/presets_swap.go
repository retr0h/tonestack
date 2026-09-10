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
	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
)

var presetsSwapOptions sdk.Edit

// presetsSwapCmd represents the presets swap command.
var presetsSwapCmd = &cobra.Command{
	Use:   "swap",
	Short: "Exchange two slots",
	Long: `Exchange two presets.

This is what moving a preset means here. Leaving the source blank would mean
writing an empty preset, and an empty preset is not empty: it carries the
inputs, outputs, split and join a device expects, which differ by model and by
firmware. Swapping invents nothing and undoes itself when repeated.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// The operation answers with what it did; saying so is decided here,
		// which is all this command does.
		change, err := swapped(cmd)
		if err != nil {
			return err
		}

		return cli.Change(cmd.OutOrStdout(), change)
	},
}

func init() {
	presetsCmd.AddCommand(presetsSwapCmd)
	editFlags(presetsSwapCmd, &presetsSwapOptions)
}

// swapped exchanges two slots, on the device or in a file.
//
// No file means the device itself, which is what somebody with one plugged in
// almost always wants.
func swapped(cmd *cobra.Command) (sdk.Change, error) {
	return sdk.New().Swap(cmd.Context(), presetsSwapOptions)
}
