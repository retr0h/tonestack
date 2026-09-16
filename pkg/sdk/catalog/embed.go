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

// The generated catalogs, one per device this tool can write a preset for.
//
// They ship inside the binary so nothing about describing, validating or
// writing a preset needs HX Edit installed. Generating them does, see
// docs/catalog.md, but that happens once per Line 6 release, on one machine,
// not on every machine that runs this.
//
// Gzipped because it is repetitive JSON: about 1MB becomes about 74KB, so
// four of them cost roughly 300KB of binary.
//
// One model table produces all four. Each model names the devices that
// support it, so the same Line 6 resources filter to a different catalog per
// device.
var (
	//go:embed data/hx-stomp.json.gz
	builtIn []byte
	//go:embed data/hx-stomp-xl.json.gz
	stompXL []byte
	//go:embed data/helix-floor.json.gz
	helixFloor []byte
	//go:embed data/helix-lt.json.gz
	helixLT []byte
)

// The devices a catalog ships for, by the id a preset carries in data.device.
const (
	HXStomp    = 2162694
	HXStompXL  = 2162699
	HelixFloor = 2162689
	HelixLT    = 2162692
)

// packed is each device's catalog, still compressed.
var packed = map[int][]byte{
	HXStomp:    builtIn,
	HXStompXL:  stompXL,
	HelixFloor: helixFloor,
	HelixLT:    helixLT,
}

// BuiltIn returns the catalog compiled into this binary.
//
// The HX Stomp's. It is the device everything here was written against, the
// only one that has been written to over USB, and the only one the corpus
// statistics describe. For another, see For.
func BuiltIn() (*Catalog, error) { return decode(builtIn) }

// For returns the built-in catalog for one device.
//
// By the id a preset carries in data.device, which is also what filtered the
// model table when the catalog was generated.
//
// Only an HX Stomp has been checked against real hardware. The other three are
// read from Line 6's own files and describe devices nobody here has written
// to.
func For(
	device int,
) (*Catalog, error) {
	body, ok := packed[device]
	if !ok {
		return nil, &NoDeviceError{Device: device}
	}

	return decode(body)
}

// decode reads a gzipped catalog.
//
// Separate from BuiltIn so a corrupted archive can be exercised. The embedded
// copy is a compile-time constant and cannot be damaged at run time, but a
// build that shipped a truncated one should say so rather than panic.
func decode(
	packed []byte,
) (*Catalog, error) {
	zr, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return nil, fmt.Errorf("opening the built-in catalog: %w", err)
	}

	// Close returns only an error a read already hit, and the read is checked.
	defer func() { _ = zr.Close() }()

	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("reading the built-in catalog: %w", err)
	}

	return Load(bytes.NewReader(raw))
}
