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
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// errKindWithoutFrom refuses --kind on a recipe that is not a copy.
var errKindWithoutFrom = errors.New(
	"--kind says what a copy is attributed to, so it needs --from")

var (
	recipesNewOptions sdk.NewRecipe
	recipesNewFrom    string
	recipesNewKind    string
	recipesNewCatalog string
)

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
departs from another, such as one song played differently from the rest.

The recipe is written to your own recipes directory, where recipes list and
presets make find it, unless --dir names another. The output says which file
it wrote.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// --kind is what a copy is attributed to. A rig scaffolded from gear
		// is always an artist, so without --from there is nothing for it to
		// change, and taking it without a word would say it had.
		if recipesNewKind != "" && recipesNewFrom == "" {
			return errKindWithoutFrom
		}

		dir := recipesDir
		if dir == "" {
			own, err := userRecipesDir()
			if err != nil {
				return err
			}

			dir = own
		}

		client := newClient(sdk.WithUserRecipes(dir), sdk.WithCatalog(recipesNewCatalog))

		made, err := scaffolded(cmd.Context(), client)
		if err != nil {
			return cli.Hint(err)
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
	f.StringVar(&recipesNewCatalog, "catalog", "",
		"a generated catalog to check against instead of the built-in one")
	f.StringVar(&recipesNewFrom, "from", "",
		"copy an existing recipe by identifier, rather than naming gear")
	f.StringVar(&recipesNewKind, "kind", "",
		"what the copy is attributed to: artist, band, song, genre or sound")
	// Fails only for a flag that does not exist, and these are defined above.
	_ = recipesNewCmd.MarkFlagRequired("id")
	// A copy takes its gear from the rig it copies, so naming any is either a
	// mistake or a misunderstanding of what a copy is.
	recipesNewCmd.MarkFlagsOneRequired("from", "amp")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "amp")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "cab")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "pedal")
	recipesNewCmd.MarkFlagsMutuallyExclusive("from", "band")
}

// scaffolded writes the recipe the flags describe: a copy with --from, or one
// naming gear without it.
func scaffolded(
	ctx context.Context,
	client *sdk.Client,
) (sdk.Scaffolded, error) {
	if recipesNewFrom == "" {
		return client.Scaffold(ctx, recipesNewOptions)
	}

	// The report names what the copy holds: the copied rig's instrument and
	// amp rather than the --instrument and --amp flags, which a copy does not
	// read, and the copied rig's name unless --name gave another.
	return client.Extend(ctx, sdk.ExtendRecipe{
		From: recipesNewFrom,
		ID:   recipesNewOptions.ID,
		Name: recipesNewOptions.Name,
		Kind: recipesNewKind,
	})
}
