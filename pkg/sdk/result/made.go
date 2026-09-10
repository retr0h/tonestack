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

package result

import "github.com/retr0h/tonestack/pkg/sdk/chain"

// Made is a preset built from a recipe.
//
// The chain matters as much as the file. A generated preset is a set of
// decisions, and a wrong amp should be visible before anybody plugs in rather
// than after.
type Made struct {
	// Chain is the signal path that was built.
	Chain chain.Chain
	// Added are the blocks nobody asked for. A recipe names an amp; a rig is
	// four or five blocks, and a choice made on the player's behalf has to be
	// visible.
	Added []Added
	// Unfamiliar are the character terms nothing defines. Said rather than
	// refused: a term moves no knob, so an unfamiliar one costs the preset
	// nothing, and refusing a build over a word would be refusing somebody
	// the right to describe a sound in their own words.
	Unfamiliar []Unfamiliar
	// Path is the file that was written.
	Path string
}

// Added is a block put in the chain that the recipe did not name.
type Added struct {
	// Name is what the block is called.
	Name string
	// Reason says why it is there.
	Reason string
	// Share is how many chains of this kind hold one, from zero to one. Zero
	// where there is no measurement rather than where nothing holds one: a
	// block substituted for gear no model emulates was not counted, and
	// reporting it as none would read as one.
	Share float64
}

// Unfamiliar is a character term nothing defines.
type Unfamiliar struct {
	// Term is the word somebody wrote.
	Term string
	// Near are the defined terms closest to it, if any are close enough to
	// be worth suggesting.
	Near []string
}
