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

var presetsShowOptions slots.ShowOptions

// presetsShowCmd represents the presets show command.
var presetsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the signal chain in one preset",
	Long: `Show what a preset actually contains.

The preset comes from either a slot in a setlist or a standalone .hlx file.
Both decode to the same chain, which is the point: what the device holds and
what this tool generates are the same kind of thing.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return slots.Show(cmd.OutOrStdout(), presetsShowOptions)
	},
}

func init() {
	presetsCmd.AddCommand(presetsShowCmd)

	f := presetsShowCmd.Flags()
	f.StringVar(
		&presetsShowOptions.Path,
		"file",
		"",
		"a .hls setlist or .hlb backup written by HX Edit",
	)
	f.StringVar(&presetsShowOptions.File, "preset", "", "a standalone .hlx preset file to read")
	f.IntVar(
		&presetsShowOptions.Setlist,
		"setlist",
		0,
		"which setlist, when the file is a backup holding several",
	)
	f.IntVar(&presetsShowOptions.Slot, "slot", 0, "which slot, from zero")
	f.StringVar(&presetsShowOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	presetsShowCmd.MarkFlagsOneRequired("file", "preset")
	presetsShowCmd.MarkFlagsMutuallyExclusive("file", "preset")
}
