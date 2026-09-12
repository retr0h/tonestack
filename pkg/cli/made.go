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
	"strings"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// Made says what was built and what it chose.
//
// Reporting the chain matters as much as writing the file. A generated preset
// is a set of decisions, and a wrong amp should be visible before anybody
// plugs in rather than after.
func Made(w io.Writer, m sdk.Made, cat *catalog.Catalog) error {
	if err := made(w, m, cat); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// made writes the parts, so a failure part way through is reported rather
// than leaving a half-written summary and a success.
func made(w io.Writer, m sdk.Made, cat *catalog.Catalog) error {
	if _, err := fmt.Fprintf(
		w, "\n%s%s\n\n", paint.Indent, paint.Title(w, m.Chain.Name),
	); err != nil {
		return err
	}

	if err := paint.Chain(w, m.Chain, cat); err != nil {
		return err
	}

	if err := added(w, m.Added); err != nil {
		return err
	}

	if err := heard(w, m.Moved); err != nil {
		return err
	}

	if err := unfamiliar(w, m.Unfamiliar); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "\n%s%s\n\n", paint.Indent, paint.Success(w, "wrote "+m.Path))

	return err
}

// added names the blocks nobody asked for, and why they are there.
func added(w io.Writer, all []sdk.Added) error {
	if len(all) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, a := range all {
		// A block drawn from the corpus can say how common it is. One
		// substituted for gear no model emulates cannot, and appending "0% of
		// chains" to it would read as a measurement.
		line := fmt.Sprintf("%s — %s", a.Name, a.Reason)
		if a.Share > 0 {
			line += fmt.Sprintf(" (%.0f%% of chains)", a.Share*100)
		}

		if _, err := fmt.Fprintf(w, "%s%s %s\n", paint.Indent, paint.Mute(w, "added"), line); err != nil {
			return err
		}
	}

	return nil
}

// heard names the knobs a word turned, and the words that turned none.
//
// Both halves, because a term that moved nothing is still something the rig
// said. Reporting only the ones that worked would read as if the rest had.
func heard(w io.Writer, all []sdk.Moved) error {
	if len(all) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, m := range all {
		line := fmt.Sprintf("%s — nothing acts on this yet", m.Term)

		switch {
		case m.Contested():
			line = fmt.Sprintf(
				"%s — another term already answered for %s, so neither moved",
				m.Term, m.Against)
		case m.Acted():
			line = fmt.Sprintf("%s — %s %.2f to %.2f", m.Term, m.Param, m.From, m.To)
		}

		if _, err := fmt.Fprintf(w, "%s%s %s\n",
			paint.Indent, paint.Mute(w, "heard"), line); err != nil {
			return err
		}
	}

	return nil
}

// unfamiliar names the character terms nothing defines.
func unfamiliar(w io.Writer, all []sdk.Unfamiliar) error {
	if len(all) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, u := range all {
		line := fmt.Sprintf("no such character term %q", u.Term)
		if len(u.Near) > 0 {
			line += " — did you mean " + strings.Join(u.Near, ", ") + "?"
		}

		if _, err := fmt.Fprintf(w, "%s%s %s\n",
			paint.Indent, paint.Mute(w, "note"), line); err != nil {
			return err
		}
	}

	return nil
}
