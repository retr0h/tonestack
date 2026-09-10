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

var presetsExportOptions sdk.Export

// presetsExportCmd represents the presets export command.
var presetsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Write one slot out as a rig",
	Long: `Pull one preset out of a setlist.

A rig by default: gear a person recognises, portable to other hardware, and the
format every other command here speaks. Compile it back with presets compile.

--as hlx writes the device's own file instead, which is a faithful copy rather
than a reading — it carries the routing and snapshots a rig models but nobody
chooses.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// The operation answers with what it wrote; saying so is decided
		// here, which is all this command does.
		written, err := exported(cmd)
		if err != nil {
			return err
		}

		return cli.Written(cmd.OutOrStdout(), written)
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
	f.Var(
		slot.NewValue(&presetsExportOptions.Slot),
		"slot",
		"which slot — a label the pedal shows such as 31A, or a number from zero",
	)
	f.StringVar(&presetsExportOptions.OutputPath, "out", "", "where to write it")
	f.StringVar((*string)(&presetsExportOptions.As), "as", "rigspec",
		"rigspec for a rig, hlx for the device's own file")
	f.StringVar(&presetsExportOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	_ = presetsExportCmd.MarkFlagRequired("slot")
	_ = presetsExportCmd.MarkFlagRequired("out")
}

// exported writes one slot out, from the device or from a file.
//
// No file means the device itself, which is what somebody with one plugged in
// almost always wants.
func exported(cmd *cobra.Command) (sdk.Written, error) {
	return sdk.New().Export(cmd.Context(), presetsExportOptions)
}
