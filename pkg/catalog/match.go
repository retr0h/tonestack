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

package catalog

import "strings"

// Matches reports whether a block is what somebody meant by a gear name.
//
// Two things make a plain comparison wrong.
//
// Line 6 print trademark symbols in their own documentation — the catalog
// holds "Klon® Centaur" and "Ampeg SVT®" — and nobody types those. They are
// stripped from both sides rather than expected.
//
// And a Line 6 original imitates nothing, so its BasedOn reads "Line 6
// Original" and the only usable handle is its name. Matching BasedOn alone
// would make every original model impossible to ask for.
func (b Block) Matches(want string) bool {
	want = normalize(want)
	if want == "" {
		return false
	}

	return strings.Contains(normalize(b.BasedOn), want) ||
		strings.Contains(normalize(b.Name), want)
}

// noise is the punctuation that appears in a manufacturer's own spelling and
// in nobody's typing.
var noise = strings.NewReplacer(
	"®", "", "™", "", "©", "", " ", " ",
)

// normalize reduces a gear name to what two people would agree it says.
func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(noise.Replace(s))), " ")
}
