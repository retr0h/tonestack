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

package cli

import (
	"fmt"
	"io"

	"github.com/retr0h/tonestack/pkg/sdk"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Change reports what a write did.
//
// The backups first, because they are what somebody needs before they need
// anything else here: a device has no undo, and the line naming the file is
// the only way back to what the slot held.
func Change(w io.Writer, c sdk.Change) error {
	if err := Kept(w, c.Kept...); err != nil {
		return err
	}

	if c.Mismatch {
		_, err := fmt.Fprintf(w, "\n%s%s\n", Indent, Info(w,
			"this preset was made for a different device; it may not load"))
		if err != nil {
			return err
		}
	}

	if c.From != nil {
		return moved(w, c)
	}

	return landed(w, c)
}

// moved says what went from one slot to another.
func moved(w io.Writer, c sdk.Change) error {
	// A device says what the destination stopped being, because there is no
	// file to go and look at. A file says where it was written, because
	// there is.
	done := fmt.Sprintf("%s, replacing %s", c.Action, c.Replaced)
	if !c.OnDevice() {
		done = fmt.Sprintf("%s, wrote %s", c.Action, c.Path)
	}

	_, err := fmt.Fprintf(w, "\n%s%s %s %s %s %s\n\n%s%s\n\n",
		Indent,
		Accent(w, slotpkg.Label(c.From.Slot)), c.From.Name,
		Mute(w, "→"),
		Accent(w, slotpkg.Label(c.To.Slot)), c.To.Name,
		Indent, Success(w, done))

	return err
}

// landed says what arrived in one slot, with nowhere it came from.
func landed(w io.Writer, c sdk.Change) error {
	// Loading a preset writes nothing, so there is nothing to say about
	// what the slot held: it still holds it.
	if c.Action == sdk.Selected {
		_, err := fmt.Fprintf(w, "\n%s%s %s\n\n%s%s\n\n",
			Indent, Accent(w, slotpkg.Label(c.To.Slot)), c.To.Name,
			Indent, Success(w, "loaded"))

		return err
	}

	if c.OnDevice() {
		_, err := fmt.Fprintf(w, "\n%s%s %s %s\n\n%s%s\n\n",
			Indent, Accent(w, c.To.Name), Mute(w, "→"),
			Accent(w, slotpkg.Label(c.To.Slot)),
			Indent, Success(w, string(c.Action)))

		return err
	}

	_, err := fmt.Fprintf(w, "\n%s%s %s %s %s\n\n%s%s\n\n",
		Indent, Accent(w, slotpkg.Label(c.To.Slot)), c.To.Name,
		Mute(w, "replaced"), c.Replaced,
		Indent, Success(w, "wrote "+c.Path))

	return err
}

// Kept names the backups a write made before it overwrote anything.
func Kept(w io.Writer, paths ...string) error {
	for _, path := range paths {
		if path == "" {
			continue
		}

		if _, err := fmt.Fprintf(w, "\n%s%s %s",
			Indent, Mute(w, "kept"), path); err != nil {
			return err
		}
	}

	return nil
}

// Built reports a preset compiled from a rig.
func Built(w io.Writer, b sdk.Built) error {
	_, err := fmt.Fprintf(w, "\n%s%s  %s\n\n%s%s\n\n",
		Indent, Title(w, b.Name),
		Mute(w, fmt.Sprintf("%s in the chain", Plural(b.Blocks, "block"))),
		Indent, Success(w, "wrote "+b.Path))

	return err
}
