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
	"fmt"
	"io"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/setlist"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// EditOptions says which two slots to act on and where to put the result.
//
// The result always goes to a new file. These files are device backups, and
// overwriting one by default would make a mistyped slot number destroy the
// only copy of what the hardware holds.
type EditOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// Path is the .hls or .hlb file to read.
	Path string
	// FromSetlist and FromSlot address the source.
	FromSetlist int
	FromSlot    int
	// ToSetlist and ToSlot address the destination.
	ToSetlist int
	ToSlot    int
	// OutputPath is where the edited file is written.
	OutputPath string
	// BackupDir is where a device slot's old contents are kept. Empty uses
	// the state directory.
	BackupDir string
	// CatalogPath is the generated catalog, needed to read a slot before
	// replacing it.
	CatalogPath string
}

// Copy overwrites one slot with another and writes the result.
func Copy(w io.Writer, opts EditOptions) error {
	return edit(w, opts, "copied", func(d *setlist.Document, from, to setlist.Address) error {
		return d.Copy(from, to)
	})
}

// Swap exchanges two slots and writes the result.
//
// This is what moving a preset means: a slot cannot be left blank without
// writing an empty preset, and an empty preset carries routing that differs
// by device and firmware. Swapping invents nothing.
func Swap(w io.Writer, opts EditOptions) error {
	return edit(w, opts, "swapped", func(d *setlist.Document, a, b setlist.Address) error {
		return d.Swap(a, b)
	})
}

// edit applies an operation to two slots and reports what moved.
func edit(
	w io.Writer,
	opts EditOptions,
	verb string,
	apply func(*setlist.Document, setlist.Address, setlist.Address) error,
) error {
	doc, err := open(opts.Path)
	if err != nil {
		return err
	}

	from := setlist.Address{Setlist: opts.FromSetlist, Slot: opts.FromSlot}
	to := setlist.Address{Setlist: opts.ToSetlist, Slot: opts.ToSlot}

	// Read the names before the edit, so the report says what was there
	// rather than what is there now.
	fromName, err := name(doc, from)
	if err != nil {
		return err
	}

	toName, err := name(doc, to)
	if err != nil {
		return err
	}

	// Both addresses were resolved above, so the operation itself cannot
	// fail to find them.
	_ = apply(doc, from, to)

	if err := save(opts.OutputPath, doc); err != nil {
		return err
	}

	return report(w, verb, fromName, toName, from, to, opts.OutputPath)
}

// name reads the name of a slot.
func name(doc *setlist.Document, at setlist.Address) (string, error) {
	d, err := doc.Slot(at.Setlist, at.Slot)
	if err != nil {
		return "", err
	}

	return d.Meta.Name, nil
}

// report says what changed and where it was written.
func report(
	w io.Writer,
	verb, fromName, toName string,
	from, to setlist.Address,
	path string,
) error {
	_, err := fmt.Fprintf(w, "\n%s%s %s %s %s %s\n\n%s%s\n\n",
		cli.Indent,
		cli.Accent(w, slotpkg.Label(from.Slot)), fromName,
		cli.Mute(w, "→"),
		cli.Accent(w, slotpkg.Label(to.Slot)), toName,
		cli.Indent,
		cli.Success(w, fmt.Sprintf("%s, wrote %s", verb, path)),
	)

	return err
}
