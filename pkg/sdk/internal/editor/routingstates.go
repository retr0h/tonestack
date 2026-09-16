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
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// Reading what a preset wraps its chain in, for writing it to a device.
//
// The way back from routingOf. A preset names every parameter and a device
// sends them by position, so the catalog's order is what puts them back.

// RoutingStates reads a preset's routing in the shape a device takes it.
//
// held is the routing the preset being written into already carries, and it
// decides two things. Which entries exist at all, because a device lays out
// what it lays out. And how many values each one takes: a device sends fewer
// than a model has names for, an input naming seven parameters and sending
// three, so each list is capped at the length the device itself wrote rather
// than at the catalog's. Writing the catalog's full list would hand a device a
// longer array than it produced.
//
// An entry the preset does not carry is left out, which leaves the target
// holding its own.
func RoutingStates(
	doc *preset.Document,
	cat *catalog.Catalog,
	held []wire.DeviceRouting,
) []wire.Routing {
	entries := doc.Data.Tone[processorKey]
	if len(entries) == 0 {
		return nil
	}

	out := make([]wire.Routing, 0, len(held))

	for _, was := range held {
		raw, ok := entries[was.Slot]
		if !ok {
			continue
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			continue
		}

		out = append(out, routeState(was, fields, cat))
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// routeState reads one routing entry.
func routeState(
	was wire.DeviceRouting,
	fields map[string]json.RawMessage,
	cat *catalog.Catalog,
) wire.Routing {
	out := wire.Routing{Slot: was.Slot}

	// Which input or output this is. A split and a join carry none: a device
	// knows which are its own.
	if key, ok := routeSelector[was.Slot]; ok {
		readInto(fields[key], &out.Select)
	}

	// Only a split and a join sit in the layout and can be switched off. The
	// device says which those are by carrying a model for them.
	if was.HasModel {
		readInto(fields[routeEnabled], &out.Enabled)
		readInto(fields[routePosition], &out.Position)
	}

	model, ok := routeModelFrom(fields)
	if !ok {
		return out
	}

	if was.HasModel {
		if number, ok := cat.SymbolNumber(model); ok {
			out.Model = &number
		}
	}

	sym, ok := symbolFor(model, cat)
	if !ok {
		return out
	}

	values := routeValues(sym, model, fields, len(was.Values), cat)
	out.Values = &values
	// What the device records beside the values it sends, which is how many
	// of them there are rather than how many the model names.
	out.Named = len(values)

	return out
}

// routeValues puts a routing entry's parameters back in the device's order.
//
// Capped at what the device sends. The values it sends are the leading ones
// of the model's own order, established against an untouched preset: an input
// names seven parameters and sends noiseGate, threshold and decay; an output
// names three and sends pan and gain. `select` is stored under its own key
// rather than among them.
func routeValues(
	sym catalog.Symbol,
	model catalog.ModelID,
	fields map[string]json.RawMessage,
	sends int,
	cat *catalog.Catalog,
) []any {
	// sends comes from the array the device itself wrote, which is never
	// longer than the list of names the model carries.
	types := typesOf(model, cat)
	out := make([]any, 0, sends)

	for _, name := range sym.Params[:sends] {
		// A parameter the preset does not name still takes its place, holding
		// what an absent value means for its kind. Position is the only thing
		// identifying a value on the wire.
		var v catalog.ParamValue

		if raw, ok := fields[name]; ok {
			// A preset somebody edited can put anything here. One value that
			// will not read leaves its parameter at nothing rather than
			// failing the whole write.
			_ = json.Unmarshal(raw, &v)
		}

		out = append(out, rawValue(v, types[name].Type))
	}

	return out
}

// routeModelFrom names the model a preset's routing entry uses.
//
// Every entry names one, including the inputs and outputs a device stores no
// model for, and it is the only thing that says which parameters the entry
// carries.
func routeModelFrom(
	fields map[string]json.RawMessage,
) (catalog.ModelID, bool) {
	raw, ok := fields[routeModel]
	if !ok {
		return "", false
	}

	var id catalog.ModelID
	if err := json.Unmarshal(raw, &id); err != nil {
		return "", false
	}

	return id, id != ""
}
