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

package chain

import (
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// ValidateBudget reports the first DSP processor whose blocks exceed the
// ceiling in lim.
//
// Bypassed blocks are counted. On Helix hardware a disabled block still
// occupies its processor, so skipping them would pass a rig that will not
// load.
//
// A block whose DSP cost carries an untrusted provenance is refused outright
// rather than counted. A guessed figure cannot support a claim that a rig fits.
func ValidateBudget(l BlockLookup, s Chain, lim Limits) error {
	if lim.Paths <= 0 {
		return &TopologyError{
			Reason: "limits declare no dsp processors",
		}
	}

	costs := make([]float64, lim.Paths)

	for _, sb := range s.Blocks {
		blk, ok := l.Block(sb.Model)
		if !ok {
			return &UnknownBlockError{Model: string(sb.Model)}
		}

		if sb.DSP < 0 || sb.DSP >= lim.Paths {
			return &TopologyError{
				Reason: fmt.Sprintf(
					"block %q is on processor %d, device has %d",
					sb.Model, sb.DSP, lim.Paths,
				),
			}
		}

		if !blk.DSP.Prov.Trusted() {
			return &catalog.BadParamError{
				Model: string(sb.Model), Key: "dsp",
				Reason: "cost is assumed and must not reach a user",
			}
		}

		cost := blk.DSP.Mono
		if blk.Stereo {
			cost = blk.DSP.Stereo
		}

		costs[sb.DSP] += cost
	}

	for chip, c := range costs {
		if c > lim.ChipCeiling {
			return &OverBudgetError{
				Chip:    chip,
				Cost:    c,
				Ceiling: lim.ChipCeiling,
			}
		}
	}

	return nil
}
