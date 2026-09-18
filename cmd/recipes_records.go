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
)

// recordsCorpus is the music corpus to read manifests from.
var recordsCorpus string

// recipesRecordsCmd represents the recipes records command.
var recipesRecordsCmd = &cobra.Command{
	Use:   "records",
	Short: "Show which records back each rig, and whether they match its era",
	Long: `Join what a rig claims to what was measured for it.

A rig says which years its gear describes. A manifest says when each measured
record was made. Both are honest on their own, and the join between them can
still be wrong: a record cut before the amplifier existed measures a different
rig, and the words derived from it describe gear the rig does not name.

Which half is wrong is a judgement nobody here can make. The rig may describe
the wrong period, or the records may be the wrong records, and only somebody
who knows the player can say which.

    tonestack recipes records --corpus resources/music/bass`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		all, err := newClient(ownRecipes(recipesDir)).Backing(cmd.Context(), recordsCorpus)
		if err != nil {
			return err
		}

		return cli.Backing(cmd.OutOrStdout(), all)
	},
}

func init() {
	recipesCmd.AddCommand(recipesRecordsCmd)

	recipesRecordsCmd.Flags().StringVar(&recordsCorpus, "corpus", "",
		"the music corpus holding one directory of records per player")
	_ = recipesRecordsCmd.MarkFlagRequired("corpus")
}
