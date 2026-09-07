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

package slots

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// chainOf turns a device's answer into a chain the rest of this speaks.
//
// The device names nothing. A block carries a number into the device's own
// model table and its parameters arrive as a bare array, so the catalog's
// symbol list is what puts names back on both — and that table is longer than
// the block list, because it holds a mono and a stereo entry for the same
// model.
func chainOf(name string, got wire.DevicePreset, cat *catalog.Catalog) (chain.Chain, error) {
	if len(cat.Symbols) == 0 {
		return chain.Chain{}, fmt.Errorf(
			"this catalog has no model table, so a preset read off the device " +
				"cannot be named: regenerate it with 'tonestack catalog generate'")
	}

	out := chain.Chain{Name: name, Blocks: make([]chain.Block, 0, len(got.Blocks))}

	cabs := 0

	for _, b := range got.Blocks {
		sym, ok := cat.Symbol(b.Model)
		if !ok {
			return chain.Chain{}, fmt.Errorf(
				"block %d names model %d, which this catalog's table of %d does "+
					"not reach: it was generated from a different release",
				b.Index, b.Model, len(cat.Symbols))
		}

		model := modelOf(sym.ID, cat)
		params := paramsOf(sym, b.Values)

		// An amp carrying a cabinet is one block to the device and two
		// entries in a preset. The cabinet is named here so the block can
		// point at it; what it holds is written alongside the routing.
		if len(b.Cab) > 0 {
			params[attrCab] = catalog.Enum(cabKey(cabs))
			cabs++
		}

		params[attrType] = catalog.Int(typeOf(model, cat, len(b.Cab) > 0))

		out.Blocks = append(out.Blocks, chain.Block{
			Model:  model,
			Params: params,
			// The device's own number, not a place in the chain. A footswitch
			// names the block it works on by this, and renumbering would
			// break the only link between the two.
			Pos:     b.Index,
			Enabled: b.Enabled,
		})
	}

	return out, nil
}

// Attributes a preset stores on a block, which a device leaves implied.
const (
	attrCab  = "@cab"
	attrType = "@type"
)

// What a preset means by a block's type.
//
// Measured over every HX Stomp preset in the corpus: an amp alone is 1 and an
// amp carrying a cabinet is 3, on 537 and 189 blocks with no exceptions. A
// cabinet is 2, and everything else is 0.
const (
	typeOther     = 0
	typeAmp       = 1
	typeCab       = 2
	typeAmpAndCab = 3
)

// typeOf says what kind of block a preset would call this.
//
// A device does not store it, because a device knows what it put there. A
// preset does, and one written without it is a preset that loads wrongly.
func typeOf(model catalog.ModelID, cat *catalog.Catalog, paired bool) int64 {
	blk, known := cat.Block(model)
	if !known {
		return typeOther
	}

	switch blk.Category {
	case catalog.CategoryAmp:
		if paired {
			return typeAmpAndCab
		}

		return typeAmp
	case catalog.CategoryCab:
		return typeCab
	default:
		return typeOther
	}
}

// cabKey names a paired cabinet the way a preset names it.
func cabKey(n int) string { return "cab" + strconv.Itoa(n) }

// modelOf resolves a device's own model name to the catalog's.
//
// A device names a mono and a stereo instance of the same model separately —
// 833 symbols cover 665 blocks — while the catalog names the model once, the
// way Line 6's own model files do. Trimming the suffix is what joins them, and
// 813 of 833 symbols land on a block that way.
//
// The rest are hardware this device does not have: a second effects loop, the
// flow inputs of a bigger Helix. Their own name is kept, which is what any
// unknown model gets, so the rig still rebuilds them exactly.
func modelOf(id catalog.ModelID, cat *catalog.Catalog) catalog.ModelID {
	if _, ok := cat.Block(id); ok {
		return id
	}

	for _, suffix := range []string{"Mono", "Stereo"} {
		trimmed := catalog.ModelID(strings.TrimSuffix(string(id), suffix))
		if trimmed == id {
			continue
		}

		if _, ok := cat.Block(trimmed); ok {
			return trimmed
		}
	}

	return id
}

// paramsOf puts names back on the values a device sent by position.
//
// A device sends fewer values than the table names when it has nothing to say
// about the rest, and more is not something it does — so the shorter of the
// two is what can be read.
func paramsOf(sym catalog.Symbol, values []any) map[string]catalog.ParamValue {
	out := make(map[string]catalog.ParamValue, len(values))

	for i, name := range sym.Params {
		if i >= len(values) {
			break
		}

		if v, ok := paramValue(values[i]); ok {
			out[name] = v
		}
	}

	return out
}

// paramValue keeps a device's value in the shape it arrived in.
//
// The wire layer has already narrowed a device's answer to a switch, a whole
// number or a fraction, which are the three things a parameter can be.
func paramValue(v any) (catalog.ParamValue, bool) {
	switch t := v.(type) {
	case bool:
		return catalog.Bool(t), true
	case int64:
		return catalog.Int(t), true
	case float64:
		return catalog.Float(t), true
	default:
		return catalog.ParamValue{}, false
	}
}
