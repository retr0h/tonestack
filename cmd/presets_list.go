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
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

var presetsListOptions sdk.Where

// presetsListAll shows the slots holding nothing.
//
// A rendering choice rather than an operation one: a device answers for every
// slot either way, and leaving the empty ones out is how somebody stops seeing
// where the gaps are.
var presetsListAll bool

// presetsListCmd represents the presets list command.
var presetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the presets a device holds",
	Long: `List what a device holds, or what a backup file holds.

With nothing else, this reads the attached device over USB. HX Edit has to be
quit first: it claims the editor interface exclusively.

With --file, it reads a backup instead, which needs no device.

Slots are labelled the way the hardware labels them, so 03B here is 03B
there.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// The operation answers with what is there; what to show of it and
		// what it looks like are decided here, which is all this command
		// does.
		listing, err := listing(cmd)
		if err != nil {
			return err
		}

		cat, err := catalog.Open(presetsListOptions.CatalogPath)
		if err != nil {
			return err
		}

		return cli.Listing(cmd.OutOrStdout(), listing, cat, presetsListAll)
	},
}

func init() {
	presetsCmd.AddCommand(presetsListCmd)

	f := presetsListCmd.Flags()
	f.StringVar(
		&presetsListOptions.Path,
		"file",
		"",
		"a .hls setlist or .hlb backup written by HX Edit",
	)
	f.IntVar(
		&presetsListOptions.Setlist,
		"setlist",
		0,
		"which setlist, when the file is a backup holding several",
	)
	f.StringVar(&presetsListOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.BoolVar(&presetsListAll, "all", false, "include empty slots")
}

// listing reads a setlist, from the device or from a file.
//
// No file means the device itself, which is what somebody with one plugged in
// almost always wants.
func listing(cmd *cobra.Command) (sdk.Listing, error) {
	return sdk.New().Presets(cmd.Context(), presetsListOptions)
}
