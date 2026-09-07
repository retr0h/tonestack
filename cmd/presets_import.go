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

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/slot"
)

var presetsImportOptions slots.ImportOptions

// presetsImportCmd represents the presets import command.
var presetsImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Put a preset file into a slot",
	Long: `Place a standalone .hlx into a slot.

This is how a generated preset reaches the hardware: import it into a backup,
then restore that backup with HX Edit. Whatever the slot held is gone, so the
result is written to a new file.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return slots.Import(cmd.OutOrStdout(), presetsImportOptions)
	},
}

func init() {
	presetsCmd.AddCommand(presetsImportCmd)

	f := presetsImportCmd.Flags()
	f.StringVar(
		&presetsImportOptions.Path,
		"file",
		"",
		"a .hls setlist or .hlb backup written by HX Edit",
	)
	f.StringVar(&presetsImportOptions.File, "preset", "", "the .hlx preset to place")
	f.IntVar(
		&presetsImportOptions.Setlist,
		"setlist",
		0,
		"which setlist, when the file is a backup holding several",
	)
	f.Var(
		slot.NewValue(&presetsImportOptions.Slot),
		"slot",
		"which slot — a label the pedal shows such as 31A, or a number from zero",
	)
	f.StringVar(&presetsImportOptions.OutputPath, "out", "", "where to write the edited setlist")
	_ = presetsImportCmd.MarkFlagRequired("preset")
	_ = presetsImportCmd.MarkFlagRequired("preset")
	_ = presetsImportCmd.MarkFlagRequired("slot")
	_ = presetsImportCmd.MarkFlagRequired("out")
}
