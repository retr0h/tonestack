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

// Package cli is how this project's output looks.
//
// One renderer per thing an operation answers with. Each takes a value from
// the SDK and a writer, and decides nothing about what was read or written —
// the operation already did that. A terminal draws a table where a service
// would write JSON, and neither has to know about the other.
//
// The visual language underneath — the theme, the table, the colours — is
// private. What is public here is the set of things a command calls, so that
// how this looks can change without breaking whatever is calling it.
package cli

import (
	"io"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
)

// How the output is coloured.
//
// Declared in the private half and named here, because the renderers and the
// theme both live down there and a type declared in this package would make
// that half import this one.
type Theme = paint.Theme

// SetTheme picks a theme by name, and says whether it knew the name.
func SetTheme(name string) bool { return paint.SetTheme(name) }

// ActiveTheme is the theme in use.
func ActiveTheme() *Theme { return paint.ActiveTheme() }

// ThemeNames is every theme there is, the default first.
func ThemeNames() []string { return paint.ThemeNames() }

// FailurePrefix is the mark that goes in front of an error line.
//
// Handed out rather than printed here, because whatever reports the error
// already knows how to print one. This is what makes that line read in this
// project's voice instead of the framework's.
func FailurePrefix(w io.Writer) string { return paint.FailurePrefix(w) }

// Banner is the heading a help page opens with.
func Banner(w io.Writer) string { return paint.Banner(w) }
