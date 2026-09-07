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

package catalog

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
)

// builtIn is the generated catalog for the device this tool targets.
//
// It ships inside the binary so nothing about describing, validating or
// writing a preset needs HX Edit installed. Generating it does — see
// docs/catalog.md — but that happens once per Line 6 release, on one machine,
// not on every machine that runs this.
//
// Gzipped because it is repetitive JSON: 1.5MB becomes about 65KB.
//
//go:embed data/hx-stomp.json.gz
var builtIn []byte

// BuiltIn returns the catalog compiled into this binary.
func BuiltIn() (*Catalog, error) { return decode(builtIn) }

// decode reads a gzipped catalog.
//
// Separate from BuiltIn so a corrupted archive can be exercised. The embedded
// copy is a compile-time constant and cannot be damaged at run time, but a
// build that shipped a truncated one should say so rather than panic.
func decode(packed []byte) (*Catalog, error) {
	zr, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return nil, fmt.Errorf("opening the built-in catalog: %w", err)
	}

	defer func() { _ = zr.Close() }()

	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("reading the built-in catalog: %w", err)
	}

	return Load(bytes.NewReader(raw))
}
