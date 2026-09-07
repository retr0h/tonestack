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
)

var presetsExportOptions slots.ExportOptions

// presetsExportCmd represents the presets export command.
var presetsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Write one slot out as a preset file",
	Long: `Pull one preset out of a setlist as a standalone .hlx.

The result is the same kind of file this tool generates, so a preset taken off
the device can be read, compared, and put back.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return slots.Export(cmd.OutOrStdout(), presetsExportOptions)
	},
}

func init() {
	presetsCmd.AddCommand(presetsExportCmd)

	f := presetsExportCmd.Flags()
	f.StringVar(
		&presetsExportOptions.Path,
		"file",
		"",
		"a .hls setlist or .hlb backup written by HX Edit",
	)
	f.IntVar(
		&presetsExportOptions.Setlist,
		"setlist",
		0,
		"which setlist, when the file is a backup holding several",
	)
	f.IntVar(&presetsExportOptions.Slot, "slot", 0, "which slot, from zero")
	f.StringVar(&presetsExportOptions.OutputPath, "out", "", "where to write the preset")
	_ = presetsExportCmd.MarkFlagRequired("file")
	_ = presetsExportCmd.MarkFlagRequired("slot")
	_ = presetsExportCmd.MarkFlagRequired("out")
}
