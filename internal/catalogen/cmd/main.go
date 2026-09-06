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
// Command catalogen writes schemas/hx-stomp.catalog.json from a local HX Edit
// installation. Run it with `just catalog`.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/internal/catalogen"
)

func main() {
	var (
		resources = flag.String("resources",
			"/Applications/Line6/HX Edit.app/Contents/Resources",
			"HX Edit's Contents/Resources directory")
		gearMap  = flag.String("gear-map", "schemas/gear-map.json", "gear map to join against")
		out      = flag.String("out", "schemas/hx-stomp.catalog.json", "where to write")
		deviceID = flag.Int("device-id", 2162694, "preset data.device value for the target")
		device   = flag.String("device", "HX Stomp", "device name")
	)

	flag.Parse()

	if err := run(*resources, *gearMap, *out, *deviceID, *device); err != nil {
		fmt.Fprintln(os.Stderr, "catalogen:", err)

		if errors.Is(err, catalogen.ErrNoResources) {
			fmt.Fprintln(os.Stderr,
				"\nThis data comes from a licensed HX Edit installation and cannot be\n"+
					"derived any other way. Install HX Edit, or keep the committed catalog.")
		}

		os.Exit(1)
	}
}

func run(resources, gearMap, out string, deviceID int, device string) error {
	c, err := catalogen.Build(catalogen.Options{
		ResourcesDir:  resources,
		GearMapPath:   gearMap,
		DeviceID:      deviceID,
		DeviceName:    device,
		SchemaVersion: 6,
	})
	if err != nil {
		return err
	}

	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding catalog: %w", err)
	}

	if err := os.WriteFile(out, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	named := 0

	for _, b := range c.Blocks {
		if b.BasedOn != "" {
			named++
		}
	}

	fmt.Printf("wrote %s: %d blocks for %s, %d mapped to real gear\n",
		out, len(c.Blocks), device, named)

	return nil
}
