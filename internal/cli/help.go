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
)

// Item is one named thing in a help listing: a command, or a flag.
type Item struct {
	Name        string
	Description string
}

// Help is everything a command's help page shows.
//
// It holds no cobra types. The command layer adapts cobra to this, so the
// layout is decided in one place and can be tested without building a
// command tree.
type Help struct {
	// Name is the full invocation, such as "tonestack presets make".
	Name string
	// Description is the long form, shown under the name.
	Description string
	// Usage is the one-line synopsis.
	Usage string
	// Commands are the subcommands, if any.
	Commands []Item
	// Flags are the flags this command accepts.
	Flags []Item
	// Footer is the closing hint, if any.
	Footer string
}

// Render writes the help page.
func (h Help) Render(w io.Writer) error {
	// The root page shows the banner instead, which already names the tool.
	if h.Name != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s\n", Indent, Title(w, h.Name)); err != nil {
			return err
		}
	}

	if h.Description != "" {
		if _, err := fmt.Fprintf(w, "\n%s\n", indented(h.Description)); err != nil {
			return err
		}
	}

	if h.Usage != "" {
		if err := block(w, "usage", [][]string{{Indent + h.Usage}}); err != nil {
			return err
		}
	}

	if err := block(w, "commands", itemRows(w, h.Commands)); err != nil {
		return err
	}

	if err := block(w, "flags", itemRows(w, h.Flags)); err != nil {
		return err
	}

	if h.Footer != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s\n", Indent, Mute(w, h.Footer)); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)

	return err
}

// block writes a heading and its rows, and nothing at all when empty.
func block(w io.Writer, title string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, "\n%s%s\n", Indent, Heading(w, title)); err != nil {
		return err
	}

	return Table(w, rows, nil)
}

// itemRows renders names beside their descriptions, indented under the
// heading they belong to.
func itemRows(w io.Writer, items []Item) [][]string {
	rows := make([][]string, 0, len(items))

	for _, it := range items {
		rows = append(rows, []string{
			Indent + Accent(w, it.Name),
			Mute(w, it.Description),
		})
	}

	return rows
}

// indented puts every line of a paragraph at the page's left margin.
func indented(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}

		lines[i] = Indent + l
	}

	return strings.Join(lines, "\n")
}
