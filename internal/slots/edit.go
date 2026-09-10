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
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/setlist"
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
func Copy(opts EditOptions) (sdk.Change, error) {
	return edit(opts, sdk.Copied, func(d *setlist.Document, from, to setlist.Address) error {
		return d.Copy(from, to)
	})
}

// Swap exchanges two slots and writes the result.
//
// This is what moving a preset means: a slot cannot be left blank without
// writing an empty preset, and an empty preset carries routing that differs
// by device and firmware. Swapping invents nothing.
func Swap(opts EditOptions) (sdk.Change, error) {
	return edit(opts, sdk.Swapped, func(d *setlist.Document, a, b setlist.Address) error {
		return d.Swap(a, b)
	})
}

// edit applies an operation to two slots and answers with what moved.
func edit(
	opts EditOptions,
	action sdk.Action,
	apply func(*setlist.Document, setlist.Address, setlist.Address) error,
) (sdk.Change, error) {
	doc, err := open(opts.Path)
	if err != nil {
		return sdk.Change{}, err
	}

	from := setlist.Address{Setlist: opts.FromSetlist, Slot: opts.FromSlot}
	to := setlist.Address{Setlist: opts.ToSetlist, Slot: opts.ToSlot}

	// Read the names before the edit, so the answer says what was there
	// rather than what is there now.
	fromName, err := name(doc, from)
	if err != nil {
		return sdk.Change{}, err
	}

	toName, err := name(doc, to)
	if err != nil {
		return sdk.Change{}, err
	}

	// Both addresses were resolved above, so the operation itself cannot
	// fail to find them.
	_ = apply(doc, from, to)

	if err := save(opts.OutputPath, doc); err != nil {
		return sdk.Change{}, err
	}

	return sdk.Change{
		Action:   action,
		From:     &sdk.At{Slot: from.Slot, Name: fromName},
		To:       sdk.At{Slot: to.Slot, Name: toName},
		Replaced: toName,
		Path:     opts.OutputPath,
	}, nil
}

// name reads the name of a slot.
func name(doc *setlist.Document, at setlist.Address) (string, error) {
	d, err := doc.Slot(at.Setlist, at.Slot)
	if err != nil {
		return "", err
	}

	return d.Meta.Name, nil
}
