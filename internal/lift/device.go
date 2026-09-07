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

package lift

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// processorPrefix marks a tone entry holding a chain rather than state.
const processorPrefix = "dsp"

// blockPrefix marks an entry inside a processor that is a chain block.
const blockPrefix = "block"

// deviceState records what a preset holds that a rig does not model.
//
// A rig describes a sound. A preset also carries footswitch assignments,
// snapshot names, the blocks a device puts either side of a chain, and
// metadata nobody documented. None of that is intent, all of it is somebody's
// work, and a rig that dropped it could not rebuild the preset it came from.
//
// Kept as raw JSON rather than decoded values, because a preset spells the
// same number more than one way — `"0.00"` and `0` are both real — and
// rewriting one as the other changes a file nobody asked to change.
func deviceState(doc *preset.Document) *riggen.DeviceState {
	out := &riggen.DeviceState{
		Id:      &doc.Data.Device,
		Format:  &doc.Version,
		Name:    &doc.Data.Meta.Name,
		Version: rawOf(doc.Data.DeviceVersion),
		Meta:    rawMap(doc.Data.Meta.Rest),
	}

	if len(doc.Meta) > 0 {
		file := doc.Meta
		out.File = &file
	}

	tone := map[string]json.RawMessage{}
	routing := map[string]json.RawMessage{}

	for key, entry := range doc.Data.Tone {
		if !strings.HasPrefix(key, processorPrefix) {
			// Not a processor, so it is state beside the chain: a controller
			// assignment, a snapshot, the global or Variax settings.
			tone[key] = mustRaw(entry)

			continue
		}

		for inner, value := range entry {
			if isBlock(inner) {
				continue
			}

			// A processor's non-block entries are its inputs, outputs, and
			// the split and join of a parallel path. Qualified by processor,
			// because each has its own.
			routing[key+"."+inner] = value
		}
	}

	assign(&out.Tone, tone)
	assign(&out.Routing, routing)

	return out
}

// restore puts a device's own state back into a preset being written.
//
// Only what the rig carries. A rig somebody typed has none of this, and the
// untouched preset underneath keeps whatever it came with — which is the
// right answer for a rig that was never lifted from anything.
func restore(doc *preset.Document, state *riggen.DeviceState) {
	if state == nil {
		return
	}

	if state.Id != nil {
		doc.Data.Device = *state.Id
	}

	if state.Format != nil {
		doc.Version = *state.Format
	}

	if state.Version != nil {
		_ = json.Unmarshal(*state.Version, &doc.Data.DeviceVersion)
	}

	if state.Name != nil {
		doc.Data.Meta.Name = *state.Name
	}

	if state.Meta != nil {
		doc.Data.Meta.Rest = *state.Meta
	}

	doc.Meta = nil
	if state.File != nil {
		doc.Meta = *state.File
	}

	restoreTone(doc, state.Tone)
	restoreRouting(doc, state.Routing)
}

// restoreTone puts back the entries that sit beside the processors.
func restoreTone(doc *preset.Document, tone *map[string]json.RawMessage) {
	// Replace rather than merge. An untouched preset carries entries of its
	// own — a Variax section, snapshots it was shipped with — and keeping
	// those beside the rig's would rebuild a preset holding things the
	// original never had.
	for key := range doc.Data.Tone {
		if !strings.HasPrefix(key, processorPrefix) {
			delete(doc.Data.Tone, key)
		}
	}

	if tone == nil {
		// A rig lifted from a preset that carried none says so by carrying
		// none. Keeping the template's would rebuild a preset holding
		// snapshots the original never had.
		return
	}

	for key, raw := range *tone {
		var entry preset.Tone
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}

		doc.Data.Tone[key] = entry
	}
}

// restoreRouting puts back what a device wraps a chain in.
//
// 98.6% of real presets carry inputs and outputs and one assembled from
// nothing carries none, so a preset rebuilt without them routes differently
// from the one it came from.
func restoreRouting(doc *preset.Document, routing *map[string]json.RawMessage) {
	// Replace rather than merge, for the same reason: a template routes a
	// signal its own way, and a rig that names no split must not inherit one.
	for key, entry := range doc.Data.Tone {
		// Processors only. A snapshot holds entries that are not blocks and
		// are not routing either, and they belong to the tone above.
		if !strings.HasPrefix(key, processorPrefix) {
			continue
		}

		for inner := range entry {
			if !isBlock(inner) {
				delete(entry, inner)
			}
		}
	}

	if routing == nil {
		return
	}

	for qualified, raw := range *routing {
		processor, inner, ok := strings.Cut(qualified, ".")
		if !ok {
			continue
		}

		if doc.Data.Tone[processor] == nil {
			doc.Data.Tone[processor] = preset.Tone{}
		}

		doc.Data.Tone[processor][inner] = raw
	}
}

// isBlock reports whether a processor entry is a chain block.
func isBlock(key string) bool {
	rest, ok := strings.CutPrefix(key, blockPrefix)
	if !ok || rest == "" {
		return false
	}

	_, err := strconv.Atoi(rest)

	return err == nil
}

// assign points a rig's optional map at one that has something in it.
func assign(target **map[string]json.RawMessage, m map[string]json.RawMessage) {
	if len(m) == 0 {
		return
	}

	*target = &m
}

// rawMap copies a preset's own raw fields, or nothing when it carries none.
func rawMap(m map[string]json.RawMessage) *map[string]json.RawMessage {
	if len(m) == 0 {
		return nil
	}

	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		out[k] = v
	}

	return &out
}

// rawOf renders a value as the JSON it will be written back as.
//
// The error is discarded because the only thing passed here is a number the
// preset already parsed, and a number always marshals.
func rawOf(v any) *json.RawMessage {
	body, _ := json.Marshal(v)
	out := json.RawMessage(body)

	return &out
}

// mustRaw renders a tone entry as raw JSON.
//
// A tone entry is raw JSON already, so re-encoding it cannot fail.
func mustRaw(t preset.Tone) json.RawMessage {
	body, _ := json.Marshal(t)

	return body
}
