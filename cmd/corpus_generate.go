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
	"github.com/retr0h/tonestack/internal/corpusgen"
)

var corpusGenerateOptions corpusgen.Options

// corpusGenerateCmd represents the corpus generate command.
var corpusGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Measure a corpus of presets into statistics",
	Long: `Reduce a directory of presets to what can be learned from them.

Runs when the corpus changes, not on every build. The result is committed and
ships in the binary, so nobody needs the presets to use what was measured.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		counted, err := corpusgen.Run(corpusGenerateOptions)
		if err != nil {
			return err
		}

		return cli.Counted(cmd.OutOrStdout(), counted)
	},
}

func init() {
	corpusCmd.AddCommand(corpusGenerateCmd)

	f := corpusGenerateCmd.Flags()
	f.StringVar(&corpusGenerateOptions.CorpusDir, "corpus",
		"resources/schemas/corpus", "directory holding presets to measure")
	f.StringVar(&corpusGenerateOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	f.StringVar(&corpusGenerateOptions.OutputPath, "out",
		"pkg/sdk/corpus/data/hx-stomp.stats.json.gz", "where to write the statistics")
	f.IntVar(&corpusGenerateOptions.MinSamples, "min-samples", 0,
		"how many values a parameter needs before its distribution is kept")
}
