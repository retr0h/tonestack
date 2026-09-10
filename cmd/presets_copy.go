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
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

var presetsCopyOptions sdk.Edit

// presetsCopyCmd represents the presets copy command.
var presetsCopyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy one slot over another",
	Long: `Copy a preset from one slot to another.

Whatever the destination held is gone, so the result is written to a new file
rather than over the one it came from. A device backup is often the only copy
of what the hardware holds.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// The operation answers with what it did; saying so is decided here,
		// which is all this command does.
		change, err := copied(cmd)
		if err != nil {
			return err
		}

		return cli.Change(cmd.OutOrStdout(), change)
	},
}

func init() {
	presetsCmd.AddCommand(presetsCopyCmd)
	editFlags(presetsCopyCmd, &presetsCopyOptions)
}

// editFlags declares the flags every two-slot edit shares.
func editFlags(c *cobra.Command, o *sdk.Edit) {
	f := c.Flags()
	f.StringVar(&o.Path, "file", "", "a .hls setlist or .hlb backup written by HX Edit")
	f.IntVar(&o.FromSetlist, "from-setlist", 0, "which setlist the source is in")
	f.Var(slot.NewValue(&o.FromSlot), "from",
		"slot to read — a label such as 31A, or a number from zero")
	f.IntVar(&o.ToSetlist, "to-setlist", 0, "which setlist the destination is in")
	f.Var(slot.NewValue(&o.ToSlot), "to",
		"slot to write — a label such as 31A, or a number from zero")
	f.StringVar(&o.OutputPath, "out", "", "where to write the edited setlist")
	f.StringVar(&o.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&o.BackupDir, "backup-dir", "",
		"where to keep what a device slot held; the state directory by default")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")
	// Editing a backup writes a new file, and editing a device writes the
	// device. So a file needs somewhere to put the result and a device does
	// not, and asking for one either way would be wrong in both directions.
	c.MarkFlagsRequiredTogether("file", "out")
}

// copied puts one slot into another, on the device or in a file.
//
// No file means the device itself, which is what somebody with one plugged in
// almost always wants.
func copied(cmd *cobra.Command) (sdk.Change, error) {
	return sdk.New().Copy(cmd.Context(), presetsCopyOptions)
}
