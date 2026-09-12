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

var catalogGenerateOptions sdk.Catalog

// catalogGenerateCmd represents the catalog generate command.
var catalogGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Rebuild the catalog from HX Edit's model definitions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		built, err := sdk.New().GenerateCatalog(catalogGenerateOptions)
		if err != nil {
			return err
		}

		return cli.Catalogued(cmd.OutOrStdout(), built)
	},
}

func init() {
	catalogCmd.AddCommand(catalogGenerateCmd)

	f := catalogGenerateCmd.Flags()
	f.StringVar(&catalogGenerateOptions.ResourcesDir, "resources",
		"/Applications/Line6/HX Edit.app/Contents/Resources",
		"HX Edit's Contents/Resources directory")
	f.StringVar(&catalogGenerateOptions.GearMapPath, "gear-map",
		"resources/schemas/gear-map.json", "gear map to join against")
	f.StringVar(&catalogGenerateOptions.OutputPath, "out",
		"pkg/sdk/catalog/data/hx-stomp.json.gz", "where to write the catalog")
	f.IntVar(&catalogGenerateOptions.DeviceID, "device-id", 2162694,
		"preset data.device value for the target device")
	f.StringVar(&catalogGenerateOptions.DeviceName, "device", "HX Stomp", "device name")
}
