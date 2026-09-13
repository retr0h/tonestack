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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Refreshed says what refreshing a catalog did.
type Refreshed struct {
	// Path is the catalog file.
	Path string
	// Skipped says which input this machine does not have, when nothing ran.
	Skipped string
	// Changed says the file was written. False when what was built matched
	// what was already there, byte for byte.
	Changed bool
	// Source says which release the catalog came from.
	Source string
	// Blocks is how many the device has.
	Blocks int
	// Named is how many of those map to real-world gear.
	Named int
}

// Refresh builds the catalog and writes it only when it differs from the one
// already there.
//
// Run by go generate on every machine, so a machine without HX Edit or the
// gear map skips rather than failing, and a machine with them leaves the
// committed file alone unless Line 6 changed something. Building the same
// inputs twice gives the same bytes, which is what makes comparing them
// honest.
func Refresh(opts Options) (Refreshed, error) {
	if missing := absent(opts); missing != "" {
		return Refreshed{Path: opts.OutputPath, Skipped: missing}, nil
	}

	if opts.SchemaVersion == 0 {
		opts.SchemaVersion = defaultSchemaVersion
	}

	if opts.SourceName == "" {
		opts.SourceName = defaultSourceName
	}

	c, err := Build(opts)
	if err != nil {
		return Refreshed{}, err
	}

	// A catalog holds no channels, functions or NaN floats, so encoding it
	// cannot fail.
	raw, _ := json.Marshal(c)
	body := compress(raw)

	out := Refreshed{Path: opts.OutputPath, Source: c.Source, Blocks: len(c.Blocks)}

	for _, b := range c.Blocks {
		if b.BasedOn != "" {
			out.Named++
		}
	}

	if was, err := os.ReadFile(opts.OutputPath); err == nil && bytes.Equal(was, body) {
		return out, nil
	}

	if err := os.WriteFile(opts.OutputPath, body, 0o600); err != nil {
		return Refreshed{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	out.Changed = true

	return out, nil
}

// absent names the first input a catalog needs that this machine lacks.
func absent(opts Options) string {
	// A pattern with no metacharacters beyond the star cannot be malformed.
	models, _ := filepath.Glob(filepath.Join(opts.ResourcesDir, "*.models"))
	if len(models) == 0 {
		return "no HX Edit at " + opts.ResourcesDir
	}

	if _, err := os.Stat(opts.GearMapPath); err != nil {
		return "no gear map at " + opts.GearMapPath + ", run just gear-map"
	}

	return ""
}
