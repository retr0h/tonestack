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

package slots

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/preset"
)

// ExportOptions says which slot to write out as a preset file.
type ExportOptions struct {
	// Path is the .hls or .hlb file to read.
	Path string
	// Setlist and Slot address the preset.
	Setlist int
	Slot    int
	// OutputPath is where the .hlx is written.
	OutputPath string
}

// Export writes one slot out as a standalone preset file.
func Export(w io.Writer, opts ExportOptions) error {
	doc, err := open(opts.Path)
	if err != nil {
		return err
	}

	data, err := doc.Slot(opts.Setlist, opts.Slot)
	if err != nil {
		return err
	}

	out := &preset.Document{
		Schema:  preset.Schema,
		Version: preset.Version,
		Data:    *data,
	}

	var buf bytes.Buffer

	// A payload that decoded encodes again.
	_ = preset.Write(&buf, out)

	if err := os.WriteFile(opts.OutputPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	_, err = fmt.Fprintf(w, "\n%s%s %s\n\n%s%s\n\n",
		cli.Indent, cli.Accent(w, position(opts.Slot)), data.Meta.Name,
		cli.Indent, cli.Success(w, "wrote "+opts.OutputPath))

	return err
}

// ImportOptions says which preset file to put in which slot.
type ImportOptions struct {
	// Path is the .hls or .hlb file to read.
	Path string
	// File is the .hlx to read.
	File string
	// Setlist and Slot address where it goes.
	Setlist int
	Slot    int
	// OutputPath is where the edited setlist is written.
	OutputPath string
}

// Import puts a standalone preset into a slot.
//
// Whatever the slot held is gone, which is why the result goes to a new file.
func Import(w io.Writer, opts ImportOptions) error {
	doc, err := open(opts.Path)
	if err != nil {
		return err
	}

	src, err := readPreset(opts.File)
	if err != nil {
		return err
	}

	dst, err := doc.Slot(opts.Setlist, opts.Slot)
	if err != nil {
		return err
	}

	replaced := dst.Meta.Name
	mismatch := dst.Device != 0 && src.Data.Device != dst.Device

	*dst = src.Data

	if err := save(opts.OutputPath, doc); err != nil {
		return err
	}

	if mismatch {
		_, err = fmt.Fprintf(w, "\n%s%s\n", cli.Indent, cli.Info(w,
			"this preset was made for a different device; it may not load"))
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "\n%s%s %s %s %s\n\n%s%s\n\n",
		cli.Indent, cli.Accent(w, position(opts.Slot)),
		src.Data.Meta.Name,
		cli.Mute(w, "replaced"), replaced,
		cli.Indent, cli.Success(w, "wrote "+opts.OutputPath))

	return err
}

// readPreset reads a standalone preset file.
func readPreset(path string) (*preset.Document, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	doc, err := preset.Read(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return doc, nil
}
