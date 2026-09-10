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

package editor

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// Turning a chain into what a device lays on its grid.
//
// The inverse of decode.go. A chain names gear the way a person does and a
// device names it by a number, so the catalog's model table is what makes
// one into the other.

// Placements turns a preset into the blocks a device grid holds.
//
// A preset rather than a chain, because an amp's cabinet is not in the
// chain. A preset stores the amp with a `@cab` naming a sibling entry, and
// that entry holds the cabinet's own settings.
func Placements(
	doc *preset.Document,
	cat *catalog.Catalog,
) ([]wire.Placement, error) {
	if len(cat.Symbols) == 0 {
		return nil, fmt.Errorf(
			"this catalog has no model table, so a chain cannot be written to " +
				"a device: regenerate it with 'tonestack catalog generate'")
	}

	c, err := doc.Spec()
	if err != nil {
		return nil, err
	}

	out := make([]wire.Placement, 0, len(c.Blocks))

	for i, b := range c.Blocks {
		model, ok := cat.SymbolNumber(b.Model)
		if !ok {
			return nil, fmt.Errorf(
				"chain entry %d names %q, which this catalog's model table of "+
					"%d does not carry: it was generated from a different release",
				i, b.Model, len(cat.Symbols))
		}

		sym, _ := cat.Symbol(model)

		p := wire.Placement{
			Position: b.Pos,
			Model:    model,
			Values:   valuesOf(sym, b.Params, typesOf(b.Model, cat), micOf(b.Attrs)),
			Named:    len(sym.Params),
			Enabled:  b.Enabled,
			CabModel: -1,
		}

		if err := held(&p, doc, b, cat); err != nil {
			return nil, fmt.Errorf("chain entry %d: %w", i, err)
		}

		p.Class = classOf(b.Model, cat, len(p.Cab) > 0)

		out = append(out, p)
	}

	return out, nil
}

// held reads the cabinet an amp carries with it, when it has one.
func held(
	p *wire.Placement,
	doc *preset.Document,
	b chain.Block,
	cat *catalog.Catalog,
) error {
	name, ok := cabNameOf(b)
	if !ok {
		return nil
	}

	entry, ok := doc.Data.Tone["dsp"+strconv.Itoa(b.DSP)][name]
	if !ok {
		return fmt.Errorf("names cabinet %q, which the preset does not hold", name)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(entry, &fields); err != nil {
		return fmt.Errorf("reading cabinet %q: %w", name, err)
	}

	var id catalog.ModelID
	if err := json.Unmarshal(fields[routeModel], &id); err != nil {
		return fmt.Errorf("cabinet %q names no model: %w", name, err)
	}

	model, ok := cat.SymbolNumber(id)
	if !ok {
		return fmt.Errorf(
			"cabinet %q names %q, which this catalog's model table does not carry",
			name, id)
	}

	sym, _ := cat.Symbol(model)

	p.Cab = valuesOf(sym, paramsOfJSON(sym, fields), typesOf(id, cat), micOf(fields))
	p.CabNamed = len(sym.Params)
	p.CabModel = model

	return nil
}

// cabNameOf reads which sibling entry holds this block's cabinet.
//
// A chain built here carries it as a parameter and one read back out of a
// preset carries it as an attribute, because `@cab` is @-prefixed and that
// is where a preset keeps those. Both are the same claim.
func cabNameOf(b chain.Block) (string, bool) {
	if name, ok := b.Params[attrCab].Enum(); ok {
		return name, true
	}

	raw, ok := b.Attrs[attrCab]
	if !ok {
		return "", false
	}

	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return "", false
	}

	return name, true
}

// paramsOfJSON reads a cabinet entry's named settings.
func paramsOfJSON(
	sym catalog.Symbol,
	fields map[string]json.RawMessage,
) map[string]catalog.ParamValue {
	out := make(map[string]catalog.ParamValue, len(sym.Params))

	for _, name := range sym.Params {
		raw, ok := fields[name]
		if !ok {
			continue
		}

		var v catalog.ParamValue
		if err := json.Unmarshal(raw, &v); err == nil {
			out[name] = v
		}
	}

	return out
}

// micOf reads the microphone a cabinet carries.
//
// A cabinet sends one more value than the model has names for, and that is
// the microphone. A preset keeps it as `@mic` rather than as a parameter, so
// it comes back as the tail of the value list or as nothing at all.
func micOf(fields map[string]json.RawMessage) []any {
	raw, ok := fields[cabMic]
	if !ok {
		return nil
	}

	// The microphone is past every named parameter, so the catalog says
	// nothing about it. A device sends 11, not 11.0, and the two are
	// different bytes.
	var v catalog.ParamValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}

	return []any{rawValue(v, "")}
}

// classOf says what a block calls itself, in the device's own numbering.
//
// Read off the captures rather than documented: every effect says 1, a
// cabinet on its own says 15, an amp with no cabinet says 17, and an amp
// carrying one says 18.
func classOf(
	model catalog.ModelID,
	cat *catalog.Catalog,
	paired bool,
) int {
	blk, known := cat.Block(model)
	if !known {
		return wire.ClassEffect
	}

	switch blk.Category {
	case catalog.CategoryAmp:
		if paired {
			return wire.ClassAmpCab
		}

		return wire.ClassAmp
	case catalog.CategoryCab:
		return wire.ClassCab
	default:
		return wire.ClassEffect
	}
}

// valuesOf puts a block's parameters back in the order a device reads them.
//
// Position is the only thing naming a value on the wire, so a parameter the
// preset does not carry still takes its place, holding a zero.
func valuesOf(
	sym catalog.Symbol,
	params map[string]catalog.ParamValue,
	types map[string]catalog.Param,
	tail []any,
) []any {
	if len(sym.Params) == 0 {
		return nil
	}

	out := make([]any, 0, len(sym.Params)+len(tail))

	for _, name := range sym.Params {
		out = append(out, rawValue(params[name], types[name].Type))
	}

	return append(out, tail...)
}

// typesOf is what the catalog says each of a model's parameters holds.
//
// A preset is JSON and JSON has one number. A cabinet's Distance of 3 is
// written `3` whether the device sent a whole number or a fraction, and only
// the catalog knows which it was. Every named parameter on every cabinet in
// the captures is a float there, and every one arrived as a float on the
// wire.
func typesOf(
	model catalog.ModelID,
	cat *catalog.Catalog,
) map[string]catalog.Param {
	blk, ok := cat.Block(model)
	if !ok {
		return nil
	}

	return blk.Params
}

// rawValue is a parameter in the shape the wire layer narrows one to.
//
// A parameter the preset does not carry arrives here as the zero ParamValue,
// which has no kind, and goes out as a zero fraction. That is what a device
// sends for a value it has nothing to say about.
func rawValue(
	v catalog.ParamValue,
	want catalog.ParamType,
) any {
	switch want {
	case catalog.ParamFloat:
		return asFloat(v)
	case catalog.ParamBool:
		b, _ := v.Bool()

		return b
	case catalog.ParamInt, catalog.ParamEnum:
		return asWhole(v)
	}

	// A model the catalog does not carry says nothing about its parameters,
	// so the value's own shape is all there is to go on.
	if b, ok := v.Bool(); ok {
		return b
	}

	if n, ok := v.Int(); ok {
		return n
	}

	f, _ := v.Float()

	return f
}

// asFloat reads a value the catalog calls a fraction.
func asFloat(v catalog.ParamValue) float64 {
	if f, ok := v.Float(); ok {
		return f
	}

	n, _ := v.Int()

	return float64(n)
}

// asWhole reads a value the catalog calls a whole number.
func asWhole(v catalog.ParamValue) int64 {
	if n, ok := v.Int(); ok {
		return n
	}

	f, _ := v.Float()

	return int64(f)
}
