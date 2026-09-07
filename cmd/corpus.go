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

import "github.com/spf13/cobra"

// corpusCmd represents the corpus command.
var corpusCmd = &cobra.Command{
	Use:   "corpus",
	Short: "Work with what real presets say about a device",
	Args:  cobra.NoArgs,
	Long: `Measure and inspect a body of presets other people made.

The catalog says what a device can do. The corpus says what people actually do
with it, which is a different question: Line 6 states a default Treble of 0.68
for an Ampeg SVT, and across every SVT measured the median is 0.845.

Nothing here is authority. It is a measurement over strangers' presets,
including their mistakes, which is why every median comes with a spread.`,
}

func init() {
	rootCmd.AddCommand(corpusCmd)
}
