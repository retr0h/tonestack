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
	"errors"
	"math"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// knobWord is one of the words a rig sets gear with, and the controls a device
// might call it.
//
// Several names per word because one manufacturer calls the same control
// different things on different models: an amplifier's loudness is Ch Vol,
// a compressor's is Level, a delay's is Mix. In order, so the name a model is
// most likely to use for the word comes first.
type knobWord struct {
	// word is what a rig says.
	word string
	// of reads it from a rig's settings.
	of func(rig.Settings) *rig.Knob
	// keys are the parameters that answer to it, best first.
	keys []string
}

// knobWords is the whole vocabulary, in the order a person reading a preset would
// meet the controls.
//
// Every name here is one the catalog carries. Alternatives that sound
// plausible and appear nowhere — Mids, Lows, Highs — are left out, because a
// name no model uses cannot resolve and only makes the error longer.
var knobWords = []knobWord{
	{
		"drive", func(s rig.Settings) *rig.Knob { return s.Drive },
		[]string{"Drive", "Gain"},
	},
	{
		"bass", func(s rig.Settings) *rig.Knob { return s.Bass },
		[]string{"Bass", "Low"},
	},
	{
		"mid", func(s rig.Settings) *rig.Knob { return s.Mid },
		[]string{"Mid"},
	},
	{
		"treble", func(s rig.Settings) *rig.Knob { return s.Treble },
		[]string{"Treble", "High"},
	},
	{
		"presence", func(s rig.Settings) *rig.Knob { return s.Presence },
		[]string{"Presence"},
	},
	{
		"level", func(s rig.Settings) *rig.Knob { return s.Level },
		[]string{"Level", "ChVol", "Master", "Volume", "Output"},
	},
	{
		"mix", func(s rig.Settings) *rig.Knob { return s.Mix },
		[]string{"Mix", "Blend"},
	},
}

// setKnobs puts what a rig said about a piece of gear onto the controls the
// model actually has.
//
// Last word, over the catalog's defaults and the corpus medians underneath:
// somebody wrote 0.47 for a reason, and a build that quietly kept the median
// would hand back a preset nobody asked for.
//
// A word the model has no control for is refused rather than dropped. The
// alternative is a rig that says drive on a cabinet and builds anyway, which
// is how `drive: 0.47` sat in the example rig doing nothing.
func setKnobs(
	params chain.Params,
	blk catalog.Block,
	set *rig.Settings,
	field string,
) error {
	if set == nil {
		return nil
	}

	out := []error(nil)

	for _, k := range knobWords {
		v := k.of(*set)
		if v == nil {
			continue
		}

		key, ok := controlFor(blk, k)
		if !ok {
			out = append(out, &NoSuchValueError{
				Field: field + "." + k.word,
				Value: k.word,
				Near:  takes(blk),
				Whole: true,
			})

			continue
		}

		params[key] = scale(blk.Params[key], float64(*v))
	}

	return errors.Join(out...)
}

// controlFor finds the control a model answers a word with.
func controlFor(
	blk catalog.Block,
	k knobWord,
) (string, bool) {
	for _, key := range k.keys {
		p, ok := blk.Params[key]
		if !ok {
			continue
		}

		// A switch or a menu is not a knob. Halfway up a three-position
		// switch is not a position, and a device handed one refuses the
		// preset rather than rounding it.
		if kind := p.Default.Type(); kind == catalog.ParamFloat || kind == catalog.ParamInt {
			return key, true
		}
	}

	return "", false
}

// takes lists the words a model has controls for.
func takes(
	blk catalog.Block,
) []string {
	out := []string(nil)

	for _, k := range knobWords {
		if _, ok := controlFor(blk, k); ok {
			out = append(out, k.word)
		}
	}

	return out
}

// scale puts a rig's 0 to 1 onto the range the control is counted in.
//
// Most of this catalog's knobs are already 0 to 1, where this changes
// nothing. The ones that are not are counted in the unit the control is in —
// hertz, decibels, milliseconds — and a rig says nothing about those.
func scale(
	p catalog.Param,
	v float64,
) catalog.ParamValue {
	at := p.Min + v*(p.Max-p.Min)

	// Rounded for an integer, because a device handed 1.5 for a control that
	// counts in whole steps refuses the preset rather than rounding it.
	if p.Default.Type() == catalog.ParamInt {
		return catalog.Int(int64(math.Round(at)))
	}

	return catalog.Float(at)
}
