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

package paint

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Indent is the left margin every block of output shares.
const Indent = "  "

// Gap separates columns.
const Gap = "  "

// Section is a titled block of tabular output.
//
// Every listing this CLI prints is one of these, so a table looks the same
// whichever command produced it.
type Section struct {
	// Title names what is being shown.
	Title string
	// Detail is metadata shown beside the title, such as a count.
	Detail string
	// Headers label the columns. Rendered uppercase.
	Headers []string
	// Rows are the data, already styled by whoever built them.
	Rows [][]string
	// Align gives the alignment per column; a short slice leaves the rest
	// left-aligned.
	Align []lipgloss.Position
	// Empty is what to say when there are no rows.
	Empty string
	// Summary is a closing line below the table.
	Summary string
}

// Render writes the section.
func (s Section) Render(w io.Writer) error {
	if err := s.title(w); err != nil {
		return err
	}

	if len(s.Rows) == 0 {
		if s.Empty == "" {
			return nil
		}

		_, err := fmt.Fprintf(w, "%s%s\n\n", Indent, Mute(w, s.Empty))

		return err
	}

	if err := Table(w, s.rows(w), s.Align); err != nil {
		return err
	}

	return s.summary(w)
}

// title writes the heading line, if there is one.
func (s Section) title(w io.Writer) error {
	if s.Title == "" {
		return nil
	}

	line := Indent + Title(w, s.Title)
	if s.Detail != "" {
		line += "  " + Mute(w, s.Detail)
	}

	_, err := fmt.Fprintf(w, "\n%s\n\n", line)

	return err
}

// rows prepends the column headings, when there are any.
func (s Section) rows(w io.Writer) [][]string {
	if len(s.Headers) == 0 {
		return s.Rows
	}

	head := make([]string, 0, len(s.Headers))
	for _, h := range s.Headers {
		head = append(head, Heading(w, h))
	}

	out := make([][]string, 0, len(s.Rows)+1)
	out = append(out, head)

	return append(out, s.Rows...)
}

// summary writes the closing line, if there is one.
func (s Section) summary(w io.Writer) error {
	if s.Summary == "" {
		_, err := fmt.Fprintln(w)

		return err
	}

	_, err := fmt.Fprintf(w, "\n%s%s\n\n", Indent, Mute(w, s.Summary))

	return err
}

// Table prints rows in aligned columns.
//
// Widths are measured with lipgloss rather than len, because a styled cell
// carries escape sequences that occupy no width on screen. Measuring bytes
// would push every column right by however much colour the cell used.
func Table(w io.Writer, rows [][]string, align []lipgloss.Position) error {
	if len(rows) == 0 {
		return nil
	}

	widths := columnWidths(rows)

	for _, row := range rows {
		cells := make([]string, 0, len(row))

		for i, cell := range row {
			// The last column needs no padding, and padding it would leave
			// trailing spaces on every line.
			if i == len(row)-1 {
				cells = append(cells, cell)

				continue
			}

			cells = append(cells, pad(cell, widths[i], alignment(align, i)))
		}

		line := strings.TrimRight(Indent+strings.Join(cells, Gap), " ")
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("writing row: %w", err)
		}
	}

	return nil
}

// columnWidths measures the widest rendered cell in each column.
func columnWidths(rows [][]string) []int {
	var widths []int

	for _, row := range rows {
		for i, cell := range row {
			for len(widths) <= i {
				widths = append(widths, 0)
			}

			if n := lipgloss.Width(cell); n > widths[i] {
				widths[i] = n
			}
		}
	}

	return widths
}

// alignment returns the alignment for a column, defaulting to left.
func alignment(align []lipgloss.Position, i int) lipgloss.Position {
	if i < len(align) {
		return align[i]
	}

	return lipgloss.Left
}

// pad widens a cell to a column width.
func pad(cell string, width int, align lipgloss.Position) string {
	fill := width - lipgloss.Width(cell)
	if fill <= 0 {
		return cell
	}

	if align == lipgloss.Right {
		return strings.Repeat(" ", fill) + cell
	}

	return cell + strings.Repeat(" ", fill)
}

// Field is one label and value in a detail view.
type Field struct {
	// Label names the value.
	Label string
	// Value is the value itself.
	Value string
	// Muted dims the value, for something absent or unremarkable.
	Muted bool
}

// Detail is a titled set of label/value lines.
//
// The counterpart to Section: a Section shows many things shallowly, a Detail
// shows one thing fully.
type Detail struct {
	// Title names the thing being shown.
	Title string
	// Subtitle sits beside the title, such as an identifier.
	Subtitle string
	// Fields are the lines.
	Fields []Field
	// Note is a closing caveat, shown in the info colour.
	Note string
}

// Render writes the detail view.
func (d Detail) Render(w io.Writer) error {
	line := Indent + Title(w, d.Title)
	if d.Subtitle != "" {
		line += "  " + Mute(w, d.Subtitle)
	}

	if _, err := fmt.Fprintf(w, "\n%s\n\n", line); err != nil {
		return err
	}

	rows := make([][]string, 0, len(d.Fields))

	for _, f := range d.Fields {
		value := f.Value
		if f.Muted {
			value = Mute(w, value)
		}

		rows = append(rows, []string{Indent + Mute(w, f.Label), value})
	}

	if err := Table(w, rows, nil); err != nil {
		return err
	}

	if d.Note != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s\n", Indent, Info(w, d.Note)); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)

	return err
}
