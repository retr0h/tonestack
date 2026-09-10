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
	"fmt"
	"os"
)

// DefaultPath is where the generated catalog lives.
const DefaultPath = "resources/schemas/hx-stomp.catalog.json"

// Open reads the catalog at path.
//
// Here rather than beside whatever draws one, because opening a catalog is
// this package's own business and every part of the library needs it.
func Open(path string) (*Catalog, error) {
	// No path means the catalog that ships in the binary, which is the case
	// for anyone who has not generated their own.
	if path == "" {
		return BuiltIn()
	}

	f, err := os.Open(path) //nolint:gosec // a path the caller named
	if err != nil {
		return nil, fmt.Errorf("opening catalog: %w", err)
	}

	defer func() { _ = f.Close() }()

	c, err := Load(f)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Files reads catalogs from disk.
//
// The value a caller gets when it says nothing about where catalogs come
// from, and the seam a test replaces when it wants to say.
type Files struct{}

// Open reads the catalog at path, or the built-in one when path is empty.
func (Files) Open(path string) (*Catalog, error) { return Open(path) }
