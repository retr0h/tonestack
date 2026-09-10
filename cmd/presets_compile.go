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
	"github.com/retr0h/tonestack/internal/slots"
)

var presetsCompileOptions slots.CompileOptions

// presetsCompileCmd represents the presets compile command.
var presetsCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Turn a rig into a preset a device will load",
	Long: `Compile a rig into the device's own format.

A rig names gear a person recognises and works on any Helix; a preset names one
manufacturer's models and works on one. This is the step between, and the only
place the device's format is written.

The chain is put into an untouched preset the device itself wrote, so the
result carries the inputs, outputs, split and join a device expects — 98.6% of
real presets have them, and one assembled without them is unlike anything the
hardware has written. Pass --template to use a particular preset as that base,
which is what makes a rig read off a device rebuild exactly.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		built, err := slots.Compile(presetsCompileOptions)
		if err != nil {
			return err
		}

		return cli.Built(cmd.OutOrStdout(), built)
	},
}

func init() {
	presetsCmd.AddCommand(presetsCompileCmd)

	f := presetsCompileCmd.Flags()
	f.StringVar(&presetsCompileOptions.RigPath, "rig", "", "the rig to compile")
	f.StringVar(&presetsCompileOptions.OutputPath, "out", "",
		"where to write the preset")
	f.StringVar(&presetsCompileOptions.TemplatePath, "template", "",
		"a preset to write the chain into, instead of an untouched one")
	f.StringVar(&presetsCompileOptions.CatalogPath, "catalog", "",
		"a generated catalog to use instead of the built-in one")
	_ = presetsCompileCmd.MarkFlagRequired("rig")
	_ = presetsCompileCmd.MarkFlagRequired("out")
}
