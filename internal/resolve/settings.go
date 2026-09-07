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

package resolve

import (
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/corpus"
)

// agreementThreshold is how tightly players must agree before the corpus
// overrules the catalog, as a share of a parameter's own range.
//
// Below it, the middle of what people do is a better answer than the factory
// default — Line 6 states 0.68 for an Ampeg SVT's Treble and the corpus
// median is 0.845. Above it there is no consensus to adopt, only an average
// of disagreement, and the catalog's default is the honest answer until a
// recipe or a person says otherwise.
const agreementThreshold = 0.15

// settings decides what every knob on a block is set to.
//
// The catalog's default is the floor: Line 6 states one for every parameter
// and it is never invalid. The corpus can raise on it, but only where players
// agree closely enough that the median means something.
func settings(b catalog.Block, stats *corpus.Stats) chain.Params {
	out := make(chain.Params, len(b.Params))

	for key, p := range b.Params {
		// A parameter with no stated default has no kind, and writing a value
		// with no kind produces a preset the device rejects.
		if p.Default.Type() == "" {
			continue
		}

		out[key] = p.Default

		if v, ok := agreed(b, key, p, stats); ok {
			out[key] = v
		}
	}

	return out
}

// agreed returns the corpus median for a parameter, when there is one and
// when players agree closely enough to be worth following.
func agreed(
	b catalog.Block,
	key string,
	p catalog.Param,
	stats *corpus.Stats,
) (catalog.ParamValue, bool) {
	if stats == nil {
		return catalog.ParamValue{}, false
	}

	d, ok := stats.Param(b.ID, key)
	if !ok {
		return catalog.ParamValue{}, false
	}

	// A switch is never averaged, however unanimous the corpus is about it:
	// the middle of an enumeration is not a value the device accepts. This
	// comes first because a switch has no range to measure agreement against.
	kind := p.Default.Type()
	if kind != catalog.ParamFloat && kind != catalog.ParamInt {
		return catalog.ParamValue{}, false
	}

	span := p.Max - p.Min
	if span <= 0 || d.Spread()/span > agreementThreshold {
		return catalog.ParamValue{}, false
	}

	// Rounding an integer matters: a device given 1.5 for a three-position
	// switch does not round it, it refuses the preset.
	if kind == catalog.ParamInt {
		return catalog.Int(int64(d.Median + 0.5)), true
	}

	return catalog.Float(d.Median), true
}
