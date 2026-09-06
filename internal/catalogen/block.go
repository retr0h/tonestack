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
package catalogen

import (
	"encoding/json"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// block converts one Line 6 model definition into a catalog block.
func block(m wireModel, family string, gear gearEntry) catalog.Block {
	b := catalog.Block{
		ID:          catalog.ModelID(m.SymbolicID),
		Name:        m.Name,
		Category:    category(family),
		BasedOn:     gear.BasedOn,
		Subcategory: gear.Subcategory,
		Params:      make(map[string]catalog.Param, len(m.Params)),
		Prov:        catalog.ProvOfficial,
	}

	if m.Stereo != nil {
		b.Stereo = *m.Stereo
	}

	if m.Load != nil {
		b.DSP = catalog.DSPCost{Mono: *m.Load, Prov: catalog.ProvOfficial}
		if m.LoadStereo != nil {
			b.DSP.Stereo = *m.LoadStereo
		}
	} else {
		// Line 6 states no cost for utility blocks. Recording it as assumed
		// keeps such a block out of any preset handed to a user, which is the
		// conservative reading and the safe one.
		b.DSP = catalog.DSPCost{Prov: catalog.ProvAssumed}
	}

	for _, p := range m.Params {
		b.Params[p.SymbolicID] = param(p)
	}

	return b
}

// param converts one Line 6 parameter definition.
//
// Min and Max are set for numeric kinds only. Line 6 records a bool's bounds
// as false and true and a string's as empty, neither of which is a range.
func param(p wireParam) catalog.Param {
	out := catalog.Param{
		Key:   p.SymbolicID,
		Label: p.Name,
		Type:  paramType(p.ValueType),
		Unit:  p.DisplayType,
		Prov:  catalog.ProvOfficial,
	}

	switch p.ValueType {
	case wireInt, wireFloat:
		out.Min = number(p.Min)
		out.Max = number(p.Max)
	case wireBool, wireString:
		// no meaningful range
	}

	out.Default = defaultValue(p)

	return out
}

// paramType maps Line 6's valueType to ours.
func paramType(v int) catalog.ParamType {
	switch v {
	case wireInt:
		return catalog.ParamInt
	case wireBool:
		return catalog.ParamBool
	case wireString:
		return catalog.ParamEnum
	default:
		return catalog.ParamFloat
	}
}

// number decodes a raw JSON number, reporting zero for anything else.
func number(raw json.RawMessage) float64 {
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0
	}

	return f
}

// defaultValue decodes a parameter's default in the kind its valueType names.
func defaultValue(p wireParam) catalog.ParamValue {
	switch p.ValueType {
	case wireBool:
		var b bool
		_ = json.Unmarshal(p.Default, &b)

		return catalog.Bool(b)
	case wireString:
		var s string
		_ = json.Unmarshal(p.Default, &s)

		return catalog.Enum(s)
	case wireInt:
		var i int64
		if err := json.Unmarshal(p.Default, &i); err == nil {
			return catalog.Int(i)
		}

		return catalog.Int(int64(number(p.Default)))
	default:
		return catalog.Float(number(p.Default))
	}
}

// category maps a .models filename to a block category.
func category(family string) catalog.Category {
	switch family {
	case "amp", "preamp":
		return catalog.CategoryAmp
	case "cab", "cabmicirs", "cabmicirswithpan":
		return catalog.CategoryCab
	case "distortion":
		return catalog.CategoryDrive
	case "compressor", "gate":
		return catalog.CategoryComp
	case "delay":
		return catalog.CategoryDelay
	case "reverb":
		return catalog.CategoryReverb
	case "eq":
		return catalog.CategoryEQ
	case "modulation":
		return catalog.CategoryMod
	default:
		return catalog.CategoryOther
	}
}
