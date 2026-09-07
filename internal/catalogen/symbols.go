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

package catalogen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// symbolFile is where HX Edit keeps a device's own model table.
const symbolFile = "Helix.sym"

// symbolEntry is one row of that table as Line 6 write it.
type symbolEntry struct {
	Symbol     string   `json:"symbol"`
	Parameters []string `json:"parameters"`
}

// readSymbols reads the table a device names its models by.
//
// A preset read over USB identifies a model by position here and sends its
// parameters as a bare array in this order, so this is what turns a device's
// answer into something with names on it.
//
// Absent on an installation too old to have it, which is not fatal: a catalog
// without it still describes what the device can do, and only reading presets
// off the hardware needs it.
func readSymbols(dir string) ([]catalog.Symbol, error) {
	body, err := os.ReadFile(
		filepath.Join(dir, symbolFile),
	) //nolint:gosec // the app's own directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading %s: %w", symbolFile, err)
	}

	var entries []symbolEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", symbolFile, err)
	}

	out := make([]catalog.Symbol, 0, len(entries))

	for _, e := range entries {
		out = append(out, catalog.Symbol{
			ID:     catalog.ModelID(e.Symbol),
			Params: e.Parameters,
		})
	}

	return out, nil
}

// controlsFile is where HX Edit keeps what a device calls its own controls.
const controlsFile = "HelixControls.json"

// ledControl names the footswitch colour list inside that file.
const ledControl = "footswitchLED"

// readLEDColours reads what a device calls its footswitch colours.
//
// A device reports the position in this list rather than the name, so without
// it a switch somebody set to violet reads as 9. Lower-cased, because it is
// written for a menu and read here as a value.
//
// Absent on an installation too old to have it, which is not fatal: only
// reading footswitches off the hardware needs it.
func readLEDColours(dir string) ([]string, error) {
	body, err := os.ReadFile(
		filepath.Join(dir, controlsFile),
	) //nolint:gosec // the app's own directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading %s: %w", controlsFile, err)
	}

	// One entry out of hundreds, and `format` means something different in
	// most of them — a printf string here, a list of ranges there. Only this
	// one is a list of names, so only this one is decoded as such.
	var controls map[string]json.RawMessage

	if err := json.Unmarshal(body, &controls); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", controlsFile, err)
	}

	var control struct {
		Format []string `json:"format"`
	}

	if err := json.Unmarshal(controls[ledControl], &control); err != nil {
		return nil, fmt.Errorf("decoding %s in %s: %w", ledControl, controlsFile, err)
	}

	out := make([]string, 0, len(control.Format))
	for _, name := range control.Format {
		out = append(out, strings.ToLower(name))
	}

	return out, nil
}
