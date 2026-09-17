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
// A footswitch carries the same numbers for the same reason, and moves the
// same way.
func Refit(
	spec rig.Spec,
	before, after []chain.Block,
) rig.Spec {
	if len(before) != len(after) {
		return spec
	}

	if spec.Controllers != nil {
		out := make([]rig.Controller, 0, len(*spec.Controllers))

		for _, c := range *spec.Controllers {
			if path, pos, ok := moved(before, after, at(c.Path, 0), c.Block); ok {
				c.Path, c.Block = &path, pos
			}

			out = append(out, c)
		}

		spec.Controllers = &out
	}

	if spec.Footswitches != nil {
		out := make([]rig.Footswitch, 0, len(*spec.Footswitches))

		for _, fs := range *spec.Footswitches {
			// A switch with no block is a switch that acts on nothing, which
			// the fit cannot have moved.
			if fs.Block != nil {
				if path, pos, ok := moved(before, after, at(fs.Path, 0), *fs.Block); ok {
					fs.Path, fs.Block = &path, &pos
				}
			}

			out = append(out, fs)
		}

		spec.Footswitches = &out
	}

	return spec
}

// moved says where the block at one address ended up.
func moved(
	before, after []chain.Block,
	path, position int,
) (int, int, bool) {
	i, ok := blockIndexAt(before, path, position)
	if !ok {
		return 0, 0, false
	}

	return after[i].DSP, after[i].Pos, true
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
