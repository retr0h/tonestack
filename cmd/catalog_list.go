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

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
)

var catalogListFilter sdk.Filter

// catalogListCmd represents the catalog list command.
var catalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the blocks this device has",
	Long: `List the blocks in the catalog, optionally narrowed.

    tonestack catalog list --subcategory bass --category amp
    tonestack catalog list --search ampeg`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		blocks, err := sdk.New().Blocks(catalogPath, catalogListFilter)
		if err != nil {
			return err
		}

		return cli.Blocks(cmd.OutOrStdout(), blocks)
	},
}

func init() {
	catalogCmd.AddCommand(catalogListCmd)

	f := catalogListCmd.Flags()
	f.StringVar(&catalogListFilter.Category, "category", "",
		"keep one kind of block: amp, cab, drive, delay, reverb, eq, mod, comp")
	f.StringVar(&catalogListFilter.Subcategory, "subcategory", "",
		"keep what Line 6 tags this way: Guitar, Bass")
	f.StringVar(&catalogListFilter.Search, "search", "",
		"keep blocks whose name or real-world gear mentions this")
}
