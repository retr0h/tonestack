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

var recipesNewOptions sdk.NewRecipe

// recipesNewCmd represents the recipes new command.
var recipesNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Scaffold a recipe",
	Long: `Write a new recipe, after checking the gear it names exists.

Every gear name is resolved against the device catalog before anything is
written. A recipe naming an amplifier no device models is otherwise only
discovered when somebody tries to build from it, and by then the name has
usually been copied somewhere else too.

--from copies an existing recipe instead, comments and citations included, and
records where it came from in extends. Nothing merges the two: the copy is a
whole rig and editing it does not touch the original. Use it for a rig that
departs from another, such as one song played differently from the rest.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		recipesNewOptions.Dir = recipesDir
		if recipesNewOptions.Dir == "" {
			recipesNewOptions.Dir = "pkg/sdk/rigs"
		}

		made, err := sdk.New().Scaffold(recipesNewOptions)
		if err != nil {
			return err
		}

		return cli.Scaffolded(cmd.OutOrStdout(), made)
	},
}

func init() {
	recipesCmd.AddCommand(recipesNewCmd)

	f := recipesNewCmd.Flags()
	f.StringVar(&recipesNewOptions.ID, "id", "",
		"identifier, and the filename stem — lower case, hyphenated")
	f.StringVar(&recipesNewOptions.Name, "name", "", "the player or style")
	f.StringVar(&recipesNewOptions.Band, "band", "", "the group, where there is one")
	f.StringVar(&recipesNewOptions.Instrument, "instrument", "guitar",
		"guitar or bass — it decides which half of the catalog is eligible")
	f.StringVar(&recipesNewOptions.Amp, "amp", "",
		"real-world amplifier, such as \"Ampeg SVT\"")
	f.StringVar(&recipesNewOptions.Cab, "cab", "",
		"real-world cabinet; omit to take the amp's own pairing")
	f.StringSliceVar(&recipesNewOptions.Pedals, "pedal", nil,
		"real-world pedal, in signal order; repeat for more")
	f.StringVar(&recipesNewOptions.CatalogPath, "catalog", "",
		"a generated catalog to check against instead of the built-in one")
	f.StringVar(&recipesNewOptions.From, "from", "",
		"copy an existing recipe by identifier, rather than naming gear")
	f.StringVar(&recipesNewOptions.Kind, "kind", "",
		"what the copy is attributed to: artist, band, song, genre or sound")
	_ = recipesNewCmd.MarkFlagRequired("id")
	// A copy takes its gear from the rig it copies, so naming any is either a
	// mistake or a misunderstanding of what a copy is.
	recipesNewCmd.MarkFlagsOneRequired("from", "amp")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "amp")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "cab")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "pedal")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "band")
}
