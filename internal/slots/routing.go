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
	"encoding/json"

	"github.com/retr0h/tonestack/pkg/catalog"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// Keys a preset stores a routing entry under. A device owns these names.
const (
	routeModel    = "@model"
	routeInput    = "@input"
	routeOutput   = "@output"
	routeEnabled  = "@enabled"
	routePosition = "@position"
)

// Which slots are inputs and which are outputs, since a preset spells the
// selector differently for each.
var routeSelector = map[string]string{
	"inputA":  routeInput,
	"inputB":  routeInput,
	"outputA": routeOutput,
	"outputB": routeOutput,
}

// routingOf turns what a device wraps a chain in into what a preset stores.
//
// The device names none of its own inputs and outputs, because it knows which
// they are. A file has to name them, and the catalog carries which models this
// device uses. The split and the join name themselves, so those come from the
// answer.
func routingOf(got wire.DevicePreset, cat *catalog.Catalog) *map[string]json.RawMessage {
	if len(got.Routing) == 0 {
		return nil
	}

	out := map[string]json.RawMessage{}

	for _, r := range got.Routing {
		model, ok := routeModelOf(r, cat)
		if !ok {
			continue
		}

		fields := map[string]any{routeModel: string(model)}

		if r.HasSelect {
			if key, ok := routeSelector[r.Slot]; ok {
				fields[key] = r.Select
			}
		}

		// A split and a join sit in the layout and can be switched off; an
		// input and an output do neither.
		if r.HasModel {
			fields[routeEnabled] = r.Enabled
			fields[routePosition] = r.Position
		}

		for name, v := range namedValues(model, r.Values, cat) {
			fields[name] = v
		}

		// A map of values that came out of MessagePack always marshals.
		body, _ := json.Marshal(fields)
		out[processorKey+"."+r.Slot] = body
	}

	if len(out) == 0 {
		return nil
	}

	return &out
}

// processorKey is the processor a device with one signal path puts everything
// on. An HX Stomp has no second, and no preset in a corpus of 714 uses one.
const processorKey = "dsp0"

// routeModelOf names the model behind one routing entry.
func routeModelOf(r wire.DeviceRouting, cat *catalog.Catalog) (catalog.ModelID, bool) {
	if r.HasModel {
		sym, ok := cat.Symbol(r.Model)

		return sym.ID, ok
	}

	switch r.Slot {
	case "inputA", "inputB":
		return cat.Flow.Input, cat.Flow.Input != ""
	case "outputA":
		return cat.Flow.OutputMain, cat.Flow.OutputMain != ""
	case "outputB":
		return cat.Flow.OutputSend, cat.Flow.OutputSend != ""
	default:
		return "", false
	}
}

// namedValues puts names on the values a device sent by position.
func namedValues(
	model catalog.ModelID,
	values []any,
	cat *catalog.Catalog,
) map[string]any {
	out := map[string]any{}

	sym, ok := symbolFor(model, cat)
	if !ok {
		return out
	}

	for i, name := range sym.Params {
		if i >= len(values) {
			break
		}

		out[name] = values[i]
	}

	return out
}

// symbolFor finds a model in the device's own table.
//
// By name rather than by number: an input carries no number, and the table is
// the only place its parameters are named.
func symbolFor(model catalog.ModelID, cat *catalog.Catalog) (catalog.Symbol, bool) {
	for _, sym := range cat.Symbols {
		if sym.ID == model {
			return sym, true
		}
	}

	return catalog.Symbol{}, false
}

// deviceStateOf records what the device wraps its chain in.
//
// Only what was read. A rig that carries a partial record would rebuild into a
// preset that routes differently from the one it came from, which is worse
// than carrying none and using an untouched preset.
func deviceStateOf(got wire.DevicePreset, cat *catalog.Catalog) *riggen.DeviceState {
	routing := routingOf(got, cat)
	if routing == nil {
		return nil
	}

	id := cat.DeviceID

	return &riggen.DeviceState{Id: &id, Routing: routing}
}
