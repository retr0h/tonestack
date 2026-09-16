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
// Command datagen refreshes the catalog this binary embeds.
//
// Run by `just generate` through the directive in generate.go. A machine
// without HX Edit or the gear map skips it, and a machine with both writes the
// catalog only when it changed.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogen"
)

// root is the repository, worked out from this file rather than from wherever
// somebody ran the command.
func root() (string, error) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot tell where this generator lives")
	}

	// pkg/sdk/internal/catalogen/datagen/main.go: five directories above the
	// one this file is in.
	dir := self
	for range 6 {
		dir = filepath.Dir(dir)
	}

	return dir, nil
}

// devices are the catalogs this binary ships, one per device the tool can
// write a preset for.
//
// The id is what a preset carries in data.device, and it is also what filters
// the model table: each model names the devices that support it, so one set of
// Line 6 resources yields a different catalog per device.
//
// The HX Stomp is first because it is the device everything here was written
// against and the only one any measured figure comes from.
var devices = []struct {
	name string
	id   int
	file string
}{
	{"HX Stomp", 2162694, "hx-stomp.json.gz"},
	{"HX Stomp XL", 2162699, "hx-stomp-xl.json.gz"},
	{"Helix Floor", 2162689, "helix-floor.json.gz"},
	{"Helix LT", 2162692, "helix-lt.json.gz"},
}

func main() {
	dir, err := root()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, d := range devices {
		r, err := catalogen.Refresh(catalogen.Options{
			ResourcesDir: catalogen.DefaultResourcesDir,
			GearMapPath:  filepath.Join(dir, "resources", "schemas", "gear-map.json"),
			DeviceName:   d.name,
			DeviceID:     d.id,
			OutputPath:   filepath.Join(dir, "pkg", "sdk", "catalog", "data", d.file),
		})

		switch {
		case err != nil:
			fmt.Fprintf(os.Stderr, "catalog %s: %v\n", d.name, err)
			os.Exit(1)
		case r.Skipped != "":
			fmt.Printf("catalog %s: skipped, %s\n", d.name, r.Skipped)
		case !r.Changed:
			fmt.Printf("catalog %s: unchanged, %d blocks from %s\n",
				d.name, r.Blocks, r.Source)
		default:
			fmt.Printf("catalog %s: wrote %d blocks from %s, %d mapped to real gear\n",
				d.name, r.Blocks, r.Source, r.Named)
		}
	}
}
