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

// Package cli holds the output helpers every command paints with: the
// theme, the tables, and the chain view.
//
// Mirrors tlock's theme system — same Theme struct, same role names, same
// lipgloss renderer plumbing — so the retr0h CLIs share one shape.
// tonestack ships one theme today; more can be added behind TONESTACK_THEME
// later without touching callers.
package cli

import (
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a six-role palette covering every place tonestack's CLI surface
// emits styled text. Roles are stable across themes so a theme swap is a
// pure recolor — no callers change.
//
//	Mute    labels, secondary metadata
//	Accent  primary highlight — names, headings, the thing being shown
//	OK      success, and values somebody verified
//	Err     failures
//	Info    warm-toned hints, and values nobody verified
//	Banner* the two banner lines — Top/Bot let themes decide whether the
//	        brand color sits above or below the implicit midline
//
// Each role is a lipgloss.Style. lipgloss handles NO_COLOR and TTY detection
// via termenv so we do not reinvent it here.
type Theme struct {
	Name      string
	Mute      lipgloss.Style
	Accent    lipgloss.Style
	OK        lipgloss.Style
	Err       lipgloss.Style
	Info      lipgloss.Style
	BannerTop lipgloss.Style
	BannerBot lipgloss.Style
}

// fg is shorthand for lipgloss.NewStyle().Foreground(...) so theme
// definitions stay scannable. Hex strings render as 24-bit truecolor when
// the terminal supports it.
func fg(c string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}

// faint uses lipgloss's dim attribute rather than a specific color so the
// muted role adapts to the user's terminal background and matches
// install.sh's `\033[0;2m` output exactly.
var faint = lipgloss.NewStyle().Faint(true)

// ThemeTube is tonestack's default.
//
// A tonestack is the tone circuit in a valve amplifier, and warm amber
// (#ffa032) is what one looks like with the lights off. Everything else is
// deliberately quiet so the accent means something when it appears.
// Truecolor, so the install banner and `--help` paint the same hue.
var ThemeTube = Theme{
	Name:      "tube",
	Mute:      faint,
	Accent:    fg("#ffa032"), // tube glow
	OK:        fg("#5ac878"),
	Err:       fg("#e05252"),
	Info:      fg("#d7a13b"),
	BannerTop: faint,
	BannerBot: fg("#ffa032"),
}

var themes = []*Theme{
	&ThemeTube,
}

var active = &ThemeTube

func init() { applyEnv(os.Getenv(themeEnv)) }

// themeEnv names the variable that selects a theme.
const themeEnv = "TONESTACK_THEME"

// applyEnv selects a theme by name, ignoring one nobody registered.
//
// An unknown name leaves the default in place rather than failing: a typo in
// a shell profile should not stop the tool running.
func applyEnv(name string) {
	if t, ok := lookupTheme(name); ok {
		active = t
	}
}

// SetTheme replaces the active theme. Returns false if name is unknown —
// callers can fall back to the default and warn.
func SetTheme(name string) bool {
	t, ok := lookupTheme(name)
	if !ok {
		return false
	}

	active = t

	return true
}

// ActiveTheme returns the currently-active theme.
func ActiveTheme() *Theme { return active }

// ThemeNames returns every registered theme name. The first element is the
// default; subsequent are alphabetical so listings are deterministic.
func ThemeNames() []string {
	out := make([]string, 0, len(themes))
	for _, t := range themes {
		out = append(out, t.Name)
	}

	// The default stays first; the rest are alphabetical so listings are
	// deterministic. Sorting an empty tail is fine, which keeps this
	// branchless while there is only one theme.
	sort.Strings(out[1:])

	return out
}

// lookupTheme finds a theme by name, case-insensitively.
func lookupTheme(name string) (*Theme, bool) {
	if name == "" {
		return nil, false
	}

	want := strings.ToLower(strings.TrimSpace(name))
	for _, t := range themes {
		if strings.EqualFold(t.Name, want) {
			return t, true
		}
	}

	return nil, false
}

// rendererFor returns a lipgloss renderer bound to w, so callers writing to
// non-stdout sinks — os.Stderr, a buffer in tests — get accurate NO_COLOR
// and TTY behavior.
func rendererFor(w io.Writer) *lipgloss.Renderer {
	if f, ok := w.(*os.File); ok {
		return lipgloss.NewRenderer(f)
	}

	return lipgloss.DefaultRenderer()
}

// render paints s in st, for the sink w.
func render(w io.Writer, st lipgloss.Style, s string) string {
	return st.Renderer(rendererFor(w)).Render(s)
}

// Mute returns s rendered as secondary text per the active theme.
func Mute(w io.Writer, s string) string { return render(w, active.Mute, s) }

// Accent returns s rendered as the brand accent color.
func Accent(w io.Writer, s string) string { return render(w, active.Accent, s) }

// OK returns s in the success color.
func OK(w io.Writer, s string) string { return render(w, active.OK, s) }

// Err returns s in the error color.
func Err(w io.Writer, s string) string { return render(w, active.Err, s) }

// Info returns s in the warm-toned info color.
func Info(w io.Writer, s string) string { return render(w, active.Info, s) }

// Title renders the name of the thing being shown.
func Title(w io.Writer, s string) string {
	return render(w, active.Accent.Bold(true), s)
}

// Heading renders a column or section heading.
func Heading(w io.Writer, s string) string {
	return render(w, active.Accent.Bold(true), strings.ToUpper(s))
}

// Banner returns the TONESTACK block-letter logo, themed via the active
// theme's BannerTop and BannerBot colors. Line-level coloring matches the
// install summary so curl|bash and `tonestack --help` look the same.
//
// The E carries a middle bar (██▄) rather than the usual █▄▄, which is
// identical to C in this alphabet and would leave TONESTACK reading with two
// of them.
func Banner(w io.Writer) string {
	const top = "▀█▀ █▀█ █▄░█ █▀▀ █▀ ▀█▀ ▄▀█ █▀▀ █▄▀"
	const bot = "░█░ █▄█ █░▀█ ██▄ ▄█ ░█░ █▀█ █▄▄ █░█"

	return render(w, active.BannerTop, top) + "\n" +
		render(w, active.BannerBot, bot) + "\n"
}

// Success renders a leading check in the OK color followed by msg. Falls
// back to a bracketed word when lipgloss decides not to color.
func Success(w io.Writer, msg string) string {
	return marked(OK(w, "✓"), "[ok]", msg)
}

// Failure mirrors Success for error one-liners.
func Failure(w io.Writer, msg string) string {
	return marked(Err(w, "✗"), "[err]", msg)
}

// marked prefixes a message with a symbol, falling back to a word when
// lipgloss decided not to colour.
//
// An uncoloured ✓ is just a character in the text with nothing to say it
// means success, so a bracketed word carries the meaning instead.
func marked(symbol, fallback, msg string) string {
	if !strings.ContainsRune(symbol, 0x1b) {
		return fallback + " " + msg
	}

	return symbol + " " + msg
}
