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

var presetsCopyOptions slots.EditOptions

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
		return slots.Copy(cmd.OutOrStdout(), presetsCopyOptions)
	},
}

func init() {
	presetsCmd.AddCommand(presetsCopyCmd)
	editFlags(presetsCopyCmd, &presetsCopyOptions)
}

// editFlags declares the flags every two-slot edit shares.
func editFlags(c *cobra.Command, o *slots.EditOptions) {
	f := c.Flags()
	f.StringVar(&o.Path, "file", "", "a .hls setlist or .hlb backup written by HX Edit")
	f.IntVar(&o.FromSetlist, "from-setlist", 0, "which setlist the source is in")
	f.Var(slot.NewValue(&o.FromSlot), "from",
		"slot to read — a label such as 31A, or a number from zero")
	f.IntVar(&o.ToSetlist, "to-setlist", 0, "which setlist the destination is in")
	f.Var(slot.NewValue(&o.ToSlot), "to",
		"slot to write — a label such as 31A, or a number from zero")
	f.StringVar(&o.OutputPath, "out", "", "where to write the edited setlist")
	_ = c.MarkFlagRequired("file")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")
	_ = c.MarkFlagRequired("out")
}
