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

// HXStompLimits returns the ceilings for a Line 6 HX Stomp.
//
// ChipCeiling is in percent, matching the units Line 6 states a block's cost
// in. An earlier revision had it as a fraction, which rejected every rig: an
// Ampeg SVT costs 26.67 and no chain fits under 0.95.
//
// The block count and chip count come from published specifications and are
// still unconfirmed against hardware. A generated preset that the device
// refuses is the most likely way these are wrong.
func HXStompLimits() Limits {
	return Limits{
		// Eight blocks, and one signal path.
		//
		// Both figures are what the corpus shows rather than what the
		// marketing says: across 714 HX Stomp presets the largest holds eight
		// blocks, and not one of them has a second path. Helix Floor presets
		// use a second path in 73% of cases, so the field is real — it just
		// does not apply to this device.
		MaxBlocks:   8,
		Paths:       1,
		ChipCeiling: 95.0,
	}
}
