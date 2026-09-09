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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// ErrNoResources reports that HX Edit's model definitions were not found.
var ErrNoResources = errors.New("hx edit resources not found")

// defaultSchemaVersion is the preset schema version a catalog records when a
// caller does not name one.
const defaultSchemaVersion = 6

// defaultSourceName names the application a catalog is generated from. The
// version is read from its bundle and appended.
const defaultSourceName = "HX Edit"

// sourceName describes where a catalog's models came from, as a release
// somebody could go and check.
func sourceName(opts Options) string {
	v := appVersion(opts.ResourcesDir)
	if v == "" {
		return ""
	}

	return opts.SourceName + " " + v
}

// valueTypes as Line 6 records them. A bool's bounds are false and true, and a
// string's are empty, so neither carries a usable range.
const (
	wireInt    = 0
	wireFloat  = 1
	wireBool   = 2
	wireString = 3
)

// Build reads Line 6's model definitions and produces a catalog for one device.
//
// Models the device does not support are excluded. Line 6 states support per
// model; a model that names no devices at all is taken as universal, which is
// how their own data reads.
func Build(opts Options) (*catalog.Catalog, error) {
	paths, err := filepath.Glob(filepath.Join(opts.ResourcesDir, "*.models"))
	if err != nil {
		return nil, fmt.Errorf("globbing model definitions: %w", err)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoResources, opts.ResourcesDir)
	}

	gear, err := loadGearMap(opts.GearMapPath)
	if err != nil {
		return nil, err
	}

	out := &catalog.Catalog{
		Device:        opts.DeviceName,
		DeviceID:      opts.DeviceID,
		SchemaVersion: opts.SchemaVersion,
		Source:        sourceName(opts),
		Blocks:        make(map[catalog.ModelID]catalog.Block),
	}

	for _, path := range paths {
		models, err := readModels(path)
		if err != nil {
			return nil, err
		}

		if flow := flowFor(models, opts.DeviceID); flow != (catalog.Flow{}) {
			out.Flow = flow
		}

		family := strings.TrimSuffix(filepath.Base(path), ".models")

		for _, m := range models {
			if !supports(m, opts.DeviceID) {
				continue
			}

			out.Blocks[catalog.ModelID(m.SymbolicID)] = block(m, family, gear[m.SymbolicID])
		}
	}

	if out.Symbols, err = readSymbols(opts.ResourcesDir); err != nil {
		return nil, err
	}

	if out.LEDColours, err = readLEDColours(opts.ResourcesDir); err != nil {
		return nil, err
	}

	if len(out.Blocks) == 0 {
		return nil, fmt.Errorf("%w: no models support device %d", ErrNoResources, opts.DeviceID)
	}

	return out, nil
}

// readModels decodes one .models file.
func readModels(path string) ([]wireModel, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // a path this program globbed
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}

	var models []wireModel
	if err := json.Unmarshal(raw, &models); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", filepath.Base(path), err)
	}

	return models, nil
}

// loadGearMap reads what each model emulates.
//
// Naming no map is a choice: the catalog is still usable, only unable to
// answer a request naming real gear. Naming one that is not there is not a
// choice, and the file is gitignored — a fresh clone that swallowed this
// would generate a catalog where nothing resolves and say so only as
// "0 mapped to real gear" halfway down a report.
func loadGearMap(path string) (map[string]gearEntry, error) {
	if path == "" {
		return map[string]gearEntry{}, nil
	}

	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied build input
	if err != nil {
		return nil, fmt.Errorf("reading gear map: %w", err)
	}

	var gm gearMap
	if err := json.Unmarshal(raw, &gm); err != nil {
		return nil, fmt.Errorf("decoding gear map: %w", err)
	}

	return gm.Models, nil
}

// supports reports whether a model is available on a device.
//
// Line 6 lists supporting devices per model. A model listing none is taken as
// universal rather than unsupported — that is how their data reads, and
// excluding those would drop most of the catalog.
func supports(m wireModel, deviceID int) bool {
	if len(m.Devices) == 0 {
		return true
	}

	for _, d := range m.Devices {
		if d.ID == deviceID {
			return true
		}
	}

	return false
}
