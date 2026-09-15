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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// recipesCmd represents the recipes command.
var recipesCmd = &cobra.Command{
	Use:   "recipes",
	Short: "Work with curated gear knowledge",
	Args:  cobra.NoArgs,
	Long: `A recipe says which gear a player or style uses, and how it should
sound. Recipes name real-world gear, "Ampeg SVT", never a device model
identifier, so one recipe serves every Helix device.

This is the only knowledge here that is ours. A device catalog is generated
from Line 6's files; recipes are written by people.

Your own recipes live in $XDG_DATA_HOME/tonestack/recipes, or in
~/.local/share/tonestack/recipes when that variable is unset. recipes new
writes there, and recipes list, recipes show, presets make and the MCP server
read them beside the built-in ones. One sharing an identifier or alias with a
built-in recipe is used in its place. --dir names another directory, which is
read in place of yours, still beside the built-in ones.`,
}

var recipesDir string

func init() {
	rootCmd.AddCommand(recipesCmd)
	recipesCmd.PersistentFlags().StringVar(&recipesDir, "dir", "",
		"a directory of recipes to use instead of yours, beside the built-in ones")
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

// ownRecipes is the option naming somebody's own recipes to read: the
// directory named, or theirs.
//
// With nowhere to keep them there are none, which is nothing to refuse a
// read over. Writing one is different, and recipes new asks for the directory
// itself.
func ownRecipes(
	named string,
) sdk.Option {
	if named != "" {
		return sdk.WithUserRecipes(named)
	}

	dir, err := userRecipesDir()
	if err != nil {
		return sdk.WithUserRecipes("")
	}

	return sdk.WithUserRecipes(dir)
}
