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
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

var presetsImportOptions slots.ImportOptions

// presetsImportCmd represents the presets import command.
var presetsImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Put a preset file into a slot",
	Long: `Place a standalone .hlx into a slot.

With no --file this writes the attached device, which is how a generated preset
reaches the hardware. The chain goes into an unused slot the device itself
wrote, so everything a chain does not describe is what the device expects to
find there.

With --file it edits an HX Edit backup instead, for working without a device
attached. Either way whatever the slot held is gone, and a device has no undo.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// No file means the device itself, which is what somebody with one
		// plugged in almost always wants.
		if presetsImportOptions.Path == "" {
			return slots.ImportDevice(
				cmd.Context(), cmd.OutOrStdout(), presetsImportOptions)
		}

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
	f.StringVar(&presetsImportOptions.BackupDir, "backup-dir", "",
		"where to keep what a device slot held; the state directory by default")
	f.StringVar(&presetsImportOptions.CatalogPath, "catalog", "",
		"a catalog to resolve models against, when writing to a device")
	_ = presetsImportCmd.MarkFlagRequired("preset")
	_ = presetsImportCmd.MarkFlagRequired("slot")

	// Importing into a backup writes a new file, and importing into a device
	// writes the device. So a file needs somewhere to put the result and a
	// device does not.
	presetsImportCmd.MarkFlagsRequiredTogether("file", "out")
}
