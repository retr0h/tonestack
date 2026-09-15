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
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// recipesCmd represents the recipes command.
var recipesCmd = &cobra.Command{
	Use:   "recipes",
	Short: "Work with curated gear knowledge",
	Args:  cobra.NoArgs,
	Long: `A recipe says which gear a player or style uses, and how it should
sound. Recipes name real-world gear — "Ampeg SVT" — never a device model
identifier, so one recipe serves every Helix device.

This is the only knowledge here that is ours. A device catalog is generated
from Line 6's files; recipes are written by people.

Your own recipes live in $XDG_DATA_HOME/tonestack/recipes, or in
~/.local/share/tonestack/recipes when that variable is unset. recipes new
writes there, and recipes list, recipes show and presets make read them beside
the built-in ones. One with the same identifier as a built-in recipe is used in
its place. --dir names another directory, which is read instead of both.`,
}

var recipesDir string

func init() {
	rootCmd.AddCommand(recipesCmd)
	recipesCmd.PersistentFlags().StringVar(&recipesDir, "dir", "",
		"a directory of recipes to use instead of yours and the built-in ones")
}

// userRecipesDir is where somebody's own recipes live.
//
// Under the data directory, resolved the way backups resolve the state
// directory, rather than wherever the command happened to run: a recipe
// written into the working directory is one no later command can find.
func userRecipesDir() (string, error) {
	if data := os.Getenv("XDG_DATA_HOME"); data != "" {
		return filepath.Join(data, "tonestack", "recipes"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding somewhere to keep recipes: %w", err)
	}

	return filepath.Join(home, ".local", "share", "tonestack", "recipes"), nil
}

// recipesDirFor is the directory to read one recipe from.
//
// A directory somebody named is used as it is. Otherwise their own directory
// is used when the recipe is there, and the built-in recipes when it is not,
// so a rig of theirs is found without hiding the ones that ship.
func recipesDirFor(
	ctx context.Context,
	named string,
	id string,
) (string, error) {
	if named != "" {
		return named, nil
	}

	dir, err := userRecipesDir()
	if err != nil {
		// Nowhere to keep recipes of their own means there are none.
		return "", nil
	}

	_, err = newClient(sdk.WithRecipes(dir)).Recipe(ctx, id)

	switch {
	case err == nil:
		return dir, nil
	case errors.Is(err, sdk.ErrNoSuchRecipe):
		return "", nil
	default:
		return "", err
	}
}

// allRecipes reads the recipes a listing shows.
//
// A directory somebody named is read on its own. Otherwise their own recipes
// are read beside the built-in ones, and one of theirs replaces a built-in
// recipe with the same identifier.
func allRecipes(
	ctx context.Context,
	named string,
) (sdk.Recipes, error) {
	if named != "" {
		return newClient(sdk.WithRecipes(named)).Recipes(ctx)
	}

	shipped, err := newClient().Recipes(ctx)
	if err != nil {
		return sdk.Recipes{}, err
	}

	dir, err := userRecipesDir()
	if err != nil {
		return shipped, nil
	}

	own, err := newClient(sdk.WithRecipes(dir)).Recipes(ctx)
	if err != nil {
		return sdk.Recipes{}, err
	}

	mine := make(map[string]bool, len(own.Rigs))
	for _, spec := range own.Rigs {
		mine[spec.ID] = true
	}

	out := own.Rigs

	for _, spec := range shipped.Rigs {
		if !mine[spec.ID] {
			out = append(out, spec)
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return sdk.Recipes{Dir: dir, Rigs: out}, nil
}
