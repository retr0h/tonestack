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
	"bytes"
	"fmt"
	"io"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Reading prints the rig a preset describes.
//
// Painted for a terminal and plain for anything else, which is what makes it
// safe to redirect: the bytes are the same document either way.
func Reading(w io.Writer, r sdk.Reading) error {
	if r.Answer != nil {
		return Answer(w, *r.Answer)
	}

	// A slot holding nothing is said in the rig's own comment syntax, so a
	// redirected read is still a file something can parse.
	if r.Empty() {
		_, err := fmt.Fprintf(w, "# %s is empty\n", r.Name)

		return err
	}

	var buf bytes.Buffer

	// Reported rather than discarded. A rig that fails to validate here would
	// otherwise print nothing at all and call it success, which reads as a
	// preset holding nothing.
	if err := rig.Write(&buf, r.Rig); err != nil {
		return err
	}

	if _, err := io.WriteString(w, YAML(w, buf.String())); err != nil {
		return fmt.Errorf("writing the rig: %w", err)
	}

	return nil
}

// Answer reports a device reply that did not decode as a preset.
//
// Saying plainly what arrived beats printing a chain that would be wrong.
func Answer(w io.Writer, a sdk.Answer) error {
	return Section{
		Title:   a.Model,
		Detail:  "slot " + slotpkg.Label(a.Slot),
		Headers: []string{"the device answered"},
		Rows:    [][]string{{a.Shape}},
		Summary: fmt.Sprintf("set %s to a path to keep it", sdk.DumpEnv),
	}.Render(w)
}

// Written reports a file this wrote, and what went into it.
//
// The slot and the name as well as the path, because somebody exporting
// several in a row is checking they got the one they meant.
func Written(w io.Writer, x sdk.Written) error {
	_, err := fmt.Fprintf(w, "\n%s%s %s\n\n%s%s\n\n",
		Indent, Accent(w, slotpkg.Label(x.Slot)), x.Name,
		Indent, Success(w, "wrote "+x.Path))

	return err
}
