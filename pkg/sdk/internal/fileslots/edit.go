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

package fileslots

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/internal/setlist"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Copy overwrites one slot of a file with another and writes the result.
//
// The result always goes to a new file. These files are device backups, and
// overwriting one by default would make a mistyped slot number destroy the
// only copy of what the hardware holds.
func (*Flows) Copy(
	ctx context.Context,
	path string,
	from, to slotpkg.Address,
	out string,
) (result.Change, error) {
	return edit(ctx, path, from, to, out, result.Copied,
		func(d *setlist.Document, from, to setlist.Address) error {
			return d.Copy(from, to)
		})
}

// Swap exchanges two slots of a file and writes the result.
//
// This is what moving a preset means: a slot cannot be left blank without
// writing an empty preset, and an empty preset carries routing that differs
// by device and firmware. Swapping invents nothing.
func (*Flows) Swap(
	ctx context.Context,
	path string,
	a, b slotpkg.Address,
	out string,
) (result.Change, error) {
	return edit(ctx, path, a, b, out, result.Swapped,
		func(d *setlist.Document, a, b setlist.Address) error {
			return d.Swap(a, b)
		})
}

// edit applies an operation to two slots and answers with what moved.
func edit(
	ctx context.Context,
	path string,
	src, dst slotpkg.Address,
	out string,
	action result.Action,
	apply func(*setlist.Document, setlist.Address, setlist.Address) error,
) (result.Change, error) {
	doc, err := open(ctx, path)
	if err != nil {
		return result.Change{}, err
	}

	from := setlist.Address{Setlist: src.Setlist, Slot: src.Slot}
	to := setlist.Address{Setlist: dst.Setlist, Slot: dst.Slot}

	// Read the names before the edit, so the answer says what was there
	// rather than what is there now.
	fromName, err := name(doc, from)
	if err != nil {
		return result.Change{}, err
	}

	toName, err := name(doc, to)
	if err != nil {
		return result.Change{}, err
	}

	// Both addresses were resolved above, so the operation itself cannot
	// fail to find them.
	_ = apply(doc, from, to)

	if err := save(out, doc); err != nil {
		return result.Change{}, err
	}

	return result.Change{
		Action:   action,
		From:     &result.At{Slot: from.Slot, Name: fromName},
		To:       result.At{Slot: to.Slot, Name: toName},
		Replaced: toName,
		Path:     out,
	}, nil
}

// name reads the name of a slot.
func name(
	doc *setlist.Document,
	at setlist.Address,
) (string, error) {
	d, err := doc.Slot(at.Setlist, at.Slot)
	if err != nil {
		return "", err
	}

	return d.Meta.Name, nil
}
