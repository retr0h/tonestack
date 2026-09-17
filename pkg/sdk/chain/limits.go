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

import "strings"

// LimitsFor returns the ceilings for a device, by the name the catalog gives
// it.
//
// A device this does not know gets the HX Stomp's, which are the smallest
// here: a chain that fits the smallest device fits the others, and a build
// that quietly assumed the largest would write a preset the hardware refuses.
//
// ChipCeiling is in percent, matching the units Line 6 states a block's cost
// in. An earlier revision had it as a fraction, which rejected every rig: an
// Ampeg SVT costs 26.67 and no chain fits under 0.95.
//
// The figures are what the corpus shows rather than what the marketing says,
// and none of them is confirmed against hardware. A generated preset that the
// device refuses is the most likely way they are wrong.
func LimitsFor(
	device string,
) Limits {
	for _, l := range limits {
		if strings.EqualFold(l.device, device) {
			return l.Limits
		}
	}

	return HXStompLimits()
}

// deviceLimits is one device's ceilings and the name it is known by.
type deviceLimits struct {
	device string
	Limits
}

// limits is what each device in the corpus holds.
//
// Counted over 4,426 presets: the largest chain each device wrote, and
// whether any of its presets used a second processor. A ceiling taken from a
// handful of presets would be a ceiling on what people happened to upload, so
// where a device shares another's processing it takes that one's figures and
// says so.
var limits = []deviceLimits{
	{
		// 721 presets, the largest holding eight blocks, and not one of them
		// on a second processor.
		device: "HX Stomp",
		Limits: Limits{MaxBlocks: 8, Paths: 1, ChipCeiling: 95.0},
	},
	{
		// The same processing in a bigger box: more footswitches, the same
		// two chips. Only 12 of its presets are in the corpus and the largest
		// holds eight blocks, which agrees.
		device: "HX Stomp XL",
		Limits: Limits{MaxBlocks: 8, Paths: 1, ChipCeiling: 95.0},
	},
	{
		// 1,698 presets. The largest holds 29 blocks, 1,219 of them use the
		// second processor, and no single processor holds more than 16.
		device: "Helix Floor",
		Limits: Limits{MaxBlocks: 29, Paths: 2, ChipCeiling: 95.0},
	},
	{
		// The Helix Floor's processing in a smaller box, so it takes the
		// Floor's figures. Its own 66 presets reach 19 blocks across two
		// processors, which is a ceiling on what people uploaded rather than
		// on what the device holds.
		device: "Helix LT",
		Limits: Limits{MaxBlocks: 29, Paths: 2, ChipCeiling: 95.0},
	},
}

// HXStompLimits returns the ceilings for a Line 6 HX Stomp.
//
// The smallest device here, and what an unknown one falls back to. See
// [LimitsFor].
func HXStompLimits() Limits {
	return limits[0].Limits
}
