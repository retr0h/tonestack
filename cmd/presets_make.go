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
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

var presetsMakeOptions sdk.Make

// presetsMakeCmd represents the presets make command.
var presetsMakeCmd = &cobra.Command{
	Use:   "make",
	Short: "Build a preset from a recipe",
	Long: `Build a preset from curated knowledge.

The recipe names real-world gear; the catalog says what this device has. Every
parameter is set to what Line 6 states as its default — a recipe's character
lines do not move knobs yet.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		made, err := sdk.New().Build(presetsMakeOptions)
		if err != nil {
			return err
		}

		cat, err := catalog.Open(presetsMakeOptions.CatalogPath)
		if err != nil {
			return err
		}

		return cli.Made(cmd.OutOrStdout(), made, cat)
	},
}

func init() {
	presetsCmd.AddCommand(presetsMakeCmd)

	f := presetsMakeCmd.Flags()
	f.StringVar(&presetsMakeOptions.RecipeID, "id", "", "recipe to build from")
	f.StringVar(
		&presetsMakeOptions.RecipesDir,
		"recipes",
		"",
		"a directory of recipes to use instead of the built-in ones",
	)
	f.StringVar(&presetsMakeOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&presetsMakeOptions.StatsPath, "stats", "",
		"measured corpus statistics to use instead of the built-in ones")
	f.StringVar(&presetsMakeOptions.OutputPath, "out", "", "where to write the preset")
	_ = presetsMakeCmd.MarkFlagRequired("id")
	_ = presetsMakeCmd.MarkFlagRequired("out")
}
