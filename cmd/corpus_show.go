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

var corpusShowOptions sdk.Corpus

// corpusShowCmd represents the corpus show command.
var corpusShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show what the corpus says",
	Long: `Show how people actually set a model, or what their chains contain.

With --model, the distribution of every knob across every preset that used it.
Without one, the chain grammar: which kinds of block a guitar or bass chain
tends to hold, and which side of the amp they sit on.

The spread is the useful column. A parameter everybody sets the same way is one
this tool can be confident about; one nobody agrees on belongs to the player.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		measured, err := sdk.New().Measurements(corpusShowOptions)
		if err != nil {
			return err
		}

		return cli.Measured(cmd.OutOrStdout(), measured)
	},
}

func init() {
	corpusCmd.AddCommand(corpusShowCmd)

	f := corpusShowCmd.Flags()
	f.StringVar(&corpusShowOptions.Model, "model", "",
		"show one model's parameter distributions, by identifier")
	f.StringVar(&corpusShowOptions.Instrument, "instrument", "",
		"limit the chain grammar to guitar or bass")
	f.StringVar(&corpusShowOptions.StatsPath, "stats", "",
		"measured statistics to use instead of the built-in ones")
	f.StringVar(&corpusShowOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	corpusShowCmd.MarkFlagsMutuallyExclusive("model", "instrument")
}
