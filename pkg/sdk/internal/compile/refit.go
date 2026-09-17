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
package compile

import (
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Refit moves a rig's controller assignments onto where the fit put the
// blocks.
//
// A rig names the block a pedal moves by its place in the chain the rig
// wrote. The fit can lay that chain across two processors, and both of them
// number their blocks from zero, so the block a rig called 4 can end up being
// block 1 on the second processor. Reading the rig's number against the
// finished chain then finds the wrong block or no block at all, and the
// assignment is dropped without a word.
//
// Nothing to do on a device with one processor, where a fit moves nothing.
//
// Footswitch assignments carry the same numbers and are not written into a
// built preset yet. They will need this too.
func Refit(
	spec rig.Spec,
	before, after []chain.Block,
) rig.Spec {
	if spec.Controllers == nil || len(before) != len(after) {
		return spec
	}

	out := make([]rig.Controller, 0, len(*spec.Controllers))

	for _, c := range *spec.Controllers {
		if i, ok := blockIndexAt(before, at(c.Path, 0), c.Block); ok {
			path := after[i].DSP
			c.Path = &path
			c.Block = after[i].Pos
		}

		out = append(out, c)
	}

	spec.Controllers = &out

	return spec
}

// blockIndexAt finds where in a chain the block at a position sits.
func blockIndexAt(
	blocks []chain.Block,
	path, position int,
) (int, bool) {
	for i, b := range blocks {
		if b.DSP == path && b.Pos == position {
			return i, true
		}
	}

	return 0, false
}
