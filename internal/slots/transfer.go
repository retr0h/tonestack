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
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Format is what an export is written as.
type Format string

// The formats an export can take.
const (
	// FormatRig is a RigSpec: this project's own format, and the default.
	// Gear a person recognises, portable to other devices, and the thing
	// every other command speaks.
	FormatRig Format = "rigspec"
	// FormatPreset is the device's own file. A faithful copy, carrying the
	// routing and snapshots a rig models but a person never chooses.
	FormatPreset Format = "hlx"
)

// ExportOptions says which slot to write out as a preset file.
type ExportOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// Path is the .hls or .hlb file to read.
	Path string
	// Setlist and Slot address the preset.
	Setlist int
	Slot    int
	// OutputPath is where the result is written.
	OutputPath string
	// As is the format. Empty means a rig.
	As Format
	// CatalogPath is a catalog to name gear against. Empty means the one
	// built into this binary.
	CatalogPath string
}

// Export reads one slot out to a file.
//
// A rig by default, because that is the format this project speaks and the
// one that reads on other hardware. The device's own file is available for a
// faithful copy, which is a different thing: it carries the routing and
// snapshots a rig models but nobody chooses.
func Export(opts ExportOptions) (sdk.Written, error) {
	doc, err := open(opts.Path)
	if err != nil {
		return sdk.Written{}, err
	}

	data, err := doc.Slot(opts.Setlist, opts.Slot)
	if err != nil {
		return sdk.Written{}, err
	}

	out := &preset.Document{
		Schema:  preset.Schema,
		Version: preset.Version,
		Data:    *data,
	}

	read := sdk.Reading{Name: data.Meta.Name, Doc: out}

	// Only the device's own file was asked for, so the lift is work nobody
	// wants.
	if opts.As != FormatPreset {
		cat, err := opts.catalogs().Open(opts.CatalogPath)
		if err != nil {
			return sdk.Written{}, err
		}

		spec, err := opts.compiler().Lift(out, cat)
		if err != nil {
			return sdk.Written{}, err
		}

		read.Rig = spec
	}

	return write(read, opts)
}

// write puts a reading on disk in the format that was asked for.
//
// One place, so a slot read off the hardware and one read out of a backup
// land as the same bytes. They describe the same preset, and an export that
// depended on which end it came from would be saying otherwise.
func write(read sdk.Reading, opts ExportOptions) (sdk.Written, error) {
	var buf bytes.Buffer

	if opts.As == FormatPreset {
		// A payload that decoded encodes again.
		_ = preset.Write(&buf, read.Doc)
	} else if err := rig.Write(&buf, read.Rig); err != nil {
		return sdk.Written{}, fmt.Errorf("writing the rig: %w", err)
	}

	if err := os.WriteFile(opts.OutputPath, buf.Bytes(), 0o600); err != nil {
		return sdk.Written{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	return sdk.Written{
		Slot: opts.Slot,
		Name: read.Name,
		Path: opts.OutputPath,
	}, nil
}

// ImportOptions says which preset file to put in which slot.
type ImportOptions struct {
	// BackupDir is where the destination slot's old contents are kept.
	// Empty uses the state directory.
	BackupDir string

	// Deps are the collaborators this command works through.
	Deps

	// Path is the .hls or .hlb file to read.
	Path string
	// File is the .hlx to read.
	File string
	// Setlist and Slot address where it goes.
	Setlist int
	Slot    int
	// OutputPath is where the edited setlist is written.
	OutputPath string
	// CatalogPath is a catalog to resolve models against, when the preset is
	// going to a device. Empty means the one built into this binary.
	CatalogPath string
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
		cli.Indent, cli.Accent(w, slotpkg.Label(opts.Slot)),
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
