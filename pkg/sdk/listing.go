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

// Package sdk is what a caller holds.
//
// The types an operation hands back live here rather than beside the code
// that builds them, so that whatever renders one does not have to import
// whatever produced it. A terminal draws a table, a service writes JSON and a
// TUI keeps a cursor in it, and none of those three should know about the
// other two.
package sdk

import "github.com/retr0h/tonestack/pkg/sdk/chain"

// Listing is what a setlist holds, slot by slot.
//
// Every slot, including the ones holding nothing. A device answers for all of
// them either way, so leaving the empty ones out would decide for a caller
// what to show, and showing the gaps is how somebody finds a free slot.
type Listing struct {
	// Name is what to call this: the device that answered, or the setlist's
	// own name when it came out of a file.
	Name string
	// Slots are every position, in the order the device counts them.
	Slots []Held
}

// Used is how many slots hold anything.
func (l Listing) Used() int {
	n := 0

	for _, h := range l.Slots {
		if !h.Empty() {
			n++
		}
	}

	return n
}

// Held is one slot and what is in it.
type Held struct {
	// Slot is the position, from zero. The label a pedal prints is
	// slot.Label of it, which is a rendering decision rather than data.
	Slot int
	// Name is what the slot is called, which a device gives every slot
	// whether or not anything is in it.
	Name string
	// Blocks are the chain it holds, empty when it holds nothing.
	Blocks []chain.Block
}

// Empty says whether the slot holds a chain.
//
// A name is no guide: an untouched slot keeps the one it shipped with, and a
// slot somebody named can still have nothing in it. Only the blocks say.
func (h Held) Empty() bool { return len(h.Blocks) == 0 }
