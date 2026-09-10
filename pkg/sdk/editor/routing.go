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

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
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
	out := map[string]json.RawMessage{}

	pairedCabs(out, got, cat)

	if len(got.Routing) == 0 && len(out) == 0 {
		return nil
	}

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

// DeviceState records what the device wraps its chain in.
//
// Only what was read. A rig that carries a partial record would rebuild into a
// preset that routes differently from the one it came from, which is worse
// than carrying none and using an untouched preset.
func DeviceState(got wire.DevicePreset, cat *catalog.Catalog) *riggen.DeviceState {
	routing := routingOf(got, cat)
	if routing == nil {
		return nil
	}

	id := cat.DeviceID

	return &riggen.DeviceState{Id: &id, Routing: routing}
}

// Keys a preset stores a paired cabinet under.
const (
	cabEnabled = "@enabled"
	cabMic     = "@mic"
)

// pairedCabs writes the cabinets amps carry with them.
//
// A device stores an amp and its cabinet as one block. A preset stores the amp
// with a `@cab` pointing at a sibling entry, and that entry here. 304 of 721
// HX Stomp presets in the corpus have one, so a rig that dropped them would be
// wrong about two in five.
//
// Which cabinet is not in the answer either: an amp names the one Line 6
// voiced it with, and the catalog carries that.
func pairedCabs(
	out map[string]json.RawMessage,
	got wire.DevicePreset,
	cat *catalog.Catalog,
) {
	n := 0

	for _, b := range got.Blocks {
		if len(b.Cab) == 0 {
			continue
		}

		sym, ok := cat.Symbol(b.Model)
		if !ok {
			continue
		}

		blk, ok := cat.Block(modelOf(sym.ID, cat))
		if !ok || blk.CabLink == "" {
			continue
		}

		fields := map[string]any{
			routeModel: string(blk.CabLink),
			cabEnabled: true,
		}

		for name, v := range namedValues(blk.CabLink, b.Cab, cat) {
			fields[name] = v
		}

		// Anything past what the cabinet model has names for is the
		// microphone, which a preset stores as an attribute rather than a
		// parameter.
		if len(b.Cab) > b.CabNamed && b.CabNamed > 0 {
			fields[cabMic] = b.Cab[b.CabNamed]
		}

		// A map of values that came out of MessagePack always marshals.
		body, _ := json.Marshal(fields)
		out[processorKey+"."+cabKey(n)] = body

		n++
	}
}
