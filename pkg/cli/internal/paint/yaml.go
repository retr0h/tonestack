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
	"io"
	"strings"
)

// YAML paints a document for reading without changing what it says.
//
// A rig is the thing this project exchanges, so it is printed rather than
// summarised — and a wall of unpainted YAML is hard to read. Colour comes off
// the same theme as every table, and off entirely when the sink is not a
// terminal, so redirecting this to a file still writes the document.
//
// Line by line rather than by parsing: what goes in is already valid, and
// re-encoding it to paint it would be a second chance to change it.
func YAML(w io.Writer, body string) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		out = append(out, paintLine(w, line))
	}

	return strings.Join(out, "\n")
}

// paintLine paints one line of a document.
func paintLine(w io.Writer, line string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
	rest := line[len(indent):]

	if rest == "" {
		return line
	}

	// A comment is the one thing here nobody has to read.
	if strings.HasPrefix(rest, "#") {
		return indent + Mute(w, rest)
	}

	// A list item's marker is structure, not content.
	if marker, item, ok := strings.Cut(rest, "- "); ok && marker == "" {
		return indent + Mute(w, "- ") + paintValue(w, item, len(indent) == 0)
	}

	return indent + paintValue(w, rest, len(indent) == 0)
}

// paintValue paints a key and whatever follows it.
//
// Top-level keys carry the accent because they are what somebody scans for;
// everything nested under one is quieter, so the shape of the document reads
// before the detail does.
func paintValue(w io.Writer, s string, top bool) string {
	key, value, ok := strings.Cut(s, ":")
	if !ok {
		return s
	}

	painted := Mute(w, key+":")
	if top {
		painted = Accent(w, key+":")
	}

	return painted + swatch(w, key, value)
}

// ledKey is the field holding the colour somebody chose for a switch.
const ledKey = "led"

// lit is what a device's own colour names look like.
//
// Approximations of a pedal's LEDs, picked to be told apart on a terminal
// rather than to match a measurement. The names come from the catalog, which
// takes them from the device's own list; anything not here is printed plain
// rather than painted wrongly.
//
// `auto color` has no colour of its own — the light follows the block — so it
// is left unpainted.
var lit = map[string]int{
	"white":        0xffffff,
	"auto color":   0,
	"red":          0xff2020,
	"dark orange":  0xc75000,
	"light orange": 0xff8c00,
	"yellow":       0xffd600,
	"green":        0x30ff30,
	"turquoise":    0x00e0c0,
	"blue":         0x4080ff,
	"violet":       0xa050ff,
	"pink":         0xff60c0,
}

// swatch paints a colour name in the colour it names.
//
// Reading `violet` is fine; seeing it is the difference between reading a rig
// and recognising your own pedal.
func swatch(w io.Writer, key, value string) string {
	if strings.TrimSpace(key) != ledKey {
		return value
	}

	name := strings.TrimSpace(value)

	rgb, ok := lit[name]
	if !ok || rgb == 0 {
		return value
	}

	return " " + Swatch(w, rgb, name)
}
