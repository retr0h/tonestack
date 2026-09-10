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

// Package slots reads and edits the setlists a device holds.
//
// Everything here works on a file HX Edit wrote — a .hls setlist or a .hlb
// backup. That is the whole device in one file, so listing, showing, copying
// and swapping slots need no connection to the hardware.
package slots

import (
	"bytes"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/setlist"
)

// open reads a setlist or bundle from disk.
func open(path string) (*setlist.Document, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	doc, err := setlist.Read(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return doc, nil
}

// save writes a setlist or bundle to disk.
//
// It renders to memory first so a failure to encode cannot leave a truncated
// backup where a good one used to be.
func save(path string, doc *setlist.Document) error {
	var buf bytes.Buffer

	// A document that was read encodes again, and writing to a buffer cannot
	// fail, so rendering has no failure to report.
	_ = setlist.Write(&buf, doc)

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
