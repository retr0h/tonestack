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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
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
the built-in ones. One sharing an identifier or alias with a built-in recipe is
used in its place. --dir names another directory, which is read instead of
both.`,
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

// userRecipes reads somebody's own recipes.
//
// Dir is empty when there is nowhere to keep them, which means there are
// none. A directory that is there and cannot be read is an error, not an
// empty one: reporting it empty would hide every rig in it without a word.
func userRecipes(
	ctx context.Context,
) (sdk.Recipes, error) {
	dir, err := userRecipesDir()
	if err != nil {
		return sdk.Recipes{}, nil
	}

	return newClient(sdk.WithRecipes(dir)).Recipes(ctx)
}

// names are every way a rig can be asked for, compared as lookup compares.
func names(
	spec rig.Spec,
) []string {
	out := []string{strings.ToLower(spec.ID)}

	if spec.Aliases != nil {
		for _, a := range *spec.Aliases {
			out = append(out, strings.ToLower(a))
		}
	}

	return out
}

// answersTo reports whether asking for id finds this rig.
func answersTo(
	spec rig.Spec,
	id string,
) bool {
	for _, n := range names(spec) {
		if n == strings.ToLower(id) {
			return true
		}
	}

	return false
}

// replaces reports whether a rig of theirs stands in for a shipped one.
//
// Any name in common, identifier or alias, because asking for that name would
// otherwise find one rig through show and make and list the other.
func replaces(
	theirs rig.Spec,
	shipped rig.Spec,
) bool {
	for _, n := range names(shipped) {
		if answersTo(theirs, n) {
			return true
		}
	}

	return false
}

// recipeFor is the directory to read one recipe from, and the identifier to
// ask it for.
//
// A directory somebody named is used as it is. Otherwise a rig of theirs that
// answers to id is used, then a shipped rig that does unless one of theirs
// replaces it, so show and make pick the rig list shows.
func recipeFor(
	ctx context.Context,
	named string,
	id string,
) (string, string, error) {
	if named != "" {
		return named, id, nil
	}

	own, err := userRecipes(ctx)
	if err != nil {
		return "", "", err
	}

	if own.Dir == "" {
		return "", id, nil
	}

	for _, spec := range own.Rigs {
		if answersTo(spec, id) {
			return own.Dir, spec.ID, nil
		}
	}

	shipped, err := newClient().Recipes(ctx)
	if err != nil {
		return "", "", err
	}

	for _, spec := range shipped.Rigs {
		if !answersTo(spec, id) {
			continue
		}

		for _, theirs := range own.Rigs {
			if replaces(theirs, spec) {
				return own.Dir, theirs.ID, nil
			}
		}

		break
	}

	// A shipped rig, or none at all, which the lookup itself reports.
	return "", id, nil
}

// allRecipes reads the recipes a listing shows.
//
// A directory somebody named is read on its own. Otherwise their own recipes
// are read beside the built-in ones, and one of theirs replaces any built-in
// recipe it shares a name with.
func allRecipes(
	ctx context.Context,
	named string,
) (sdk.Recipes, error) {
	if named != "" {
		return newClient(sdk.WithRecipes(named)).Recipes(ctx)
	}

	own, err := userRecipes(ctx)
	if err != nil {
		return sdk.Recipes{}, err
	}

	shipped, err := newClient().Recipes(ctx)
	if err != nil {
		return sdk.Recipes{}, err
	}

	if own.Dir == "" {
		return shipped, nil
	}

	out := own.Rigs

	for _, spec := range shipped.Rigs {
		replaced := false

		for _, theirs := range own.Rigs {
			if replaces(theirs, spec) {
				replaced = true

				break
			}
		}

		if !replaced {
			out = append(out, spec)
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return sdk.Recipes{Dir: own.Dir, Rigs: out}, nil
}
