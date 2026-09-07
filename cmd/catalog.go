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

// catalogCmd represents the catalog command.
var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Work with the device catalog",
	Args:  cobra.NoArgs,
	Long: `The catalog says what a device can do: which blocks exist, what
parameters each accepts, their real ranges, and what each costs in DSP.

It is generated from a licensed HX Edit installation and is not redistributed.
A machine without HX Edit cannot build one.`,
}

// catalogPath is where list and show read the catalog from.
var catalogPath string

func init() {
	rootCmd.AddCommand(catalogCmd)
	catalogCmd.PersistentFlags().StringVar(&catalogPath, "catalog",
		"", "a generated catalog to read instead of the built-in one")
}
