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
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// footswitchKey is the tone entry a preset stores footswitches under.
const footswitchKey = "footswitch"

// Keys one footswitch assignment is stored under. A device owns these names.
const (
	fsIndex     = "@fs_index"
	fsLabel     = "@fs_label"
	fsColour    = "@fs_ledcolor"
	fsEnabled   = "@fs_enabled"
	fsMomentary = "@fs_momentary"
	fsPrimary   = "@fs_primary"
)

// fsModelled names the fields this package reads by name.
var fsModelled = map[string]bool{
	fsIndex: true, fsLabel: true, fsColour: true,
	fsEnabled: true, fsMomentary: true, fsPrimary: true,
}

// footswitchesOf reads what the pedal shows under each switch.
//
// A preset keys these by the block a switch acts on, inside the processor
// that block sits on. Both are carried, because a label with nothing to
// attach it to cannot be written back.
func footswitchesOf(doc *preset.Document) *[]riggen.Footswitch {
	entry, ok := doc.Data.Tone[footswitchKey]
	if !ok {
		return nil
	}

	out := []riggen.Footswitch(nil)

	for _, processor := range sorted(entry) {
		path, err := strconv.Atoi(strings.TrimPrefix(processor, processorPrefix))
		if err != nil {
			continue
		}

		var blocks map[string]json.RawMessage
		if err := json.Unmarshal(entry[processor], &blocks); err != nil {
			continue
		}

		for _, key := range sortedKeys(blocks) {
			fs, ok := footswitchOf(blocks[key], key, path)
			if !ok {
				continue
			}

			out = append(out, fs)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return &out
}

// footswitchOf reads one assignment.
func footswitchOf(raw json.RawMessage, key string, path int) (riggen.Footswitch, bool) {
	number, err := strconv.Atoi(strings.TrimPrefix(key, blockPrefix))
	if err != nil {
		return riggen.Footswitch{}, false
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return riggen.Footswitch{}, false
	}

	out := riggen.Footswitch{Block: &number}

	if path != 0 {
		out.Path = &path
	}

	decode(fields[fsIndex], &out.Switch)
	decode(fields[fsLabel], &out.Label)
	decode(fields[fsColour], &out.Colour)
	decode(fields[fsEnabled], &out.Enabled)
	decode(fields[fsMomentary], &out.Momentary)
	decode(fields[fsPrimary], &out.Primary)

	rest := map[string]json.RawMessage{}

	for name, value := range fields {
		if !fsModelled[name] {
			rest[name] = value
		}
	}

	if len(rest) > 0 {
		out.Rest = &rest
	}

	return out, true
}

// restoreFootswitches writes a rig's footswitches back as a preset stores
// them: keyed by the block each acts on, inside its processor.
func restoreFootswitches(doc *preset.Document, switches []riggen.Footswitch) {
	byProcessor := map[string]map[string]json.RawMessage{}

	for _, fs := range switches {
		if fs.Block == nil {
			continue
		}

		processor := processorPrefix + strconv.Itoa(at(fs.Path, 0))
		if byProcessor[processor] == nil {
			byProcessor[processor] = map[string]json.RawMessage{}
		}

		byProcessor[processor][blockPrefix+strconv.Itoa(*fs.Block)] = fieldsOf(fs)
	}

	if len(byProcessor) == 0 {
		return
	}

	entry := preset.Tone{}

	for processor, blocks := range byProcessor {
		// A map of raw JSON always marshals.
		body, _ := json.Marshal(blocks)
		entry[processor] = body
	}

	doc.Data.Tone[footswitchKey] = entry
}

// fieldsOf renders one assignment the way a preset stores it.
func fieldsOf(fs riggen.Footswitch) json.RawMessage {
	fields := map[string]json.RawMessage{}

	putRaw(fields, fsIndex, fs.Switch)
	putRaw(fields, fsLabel, fs.Label)
	putRaw(fields, fsColour, fs.Colour)
	putRaw(fields, fsEnabled, fs.Enabled)
	putRaw(fields, fsMomentary, fs.Momentary)
	putRaw(fields, fsPrimary, fs.Primary)

	if fs.Rest != nil {
		for name, value := range *fs.Rest {
			fields[name] = value
		}
	}

	// A map of raw JSON always marshals.
	body, _ := json.Marshal(fields)

	return body
}

// putRaw writes a field back, or writes nothing when the rig has none.
func putRaw[T any](fields map[string]json.RawMessage, key string, v *T) {
	if v == nil {
		return
	}

	// A value that came out of JSON goes back into it, so this cannot fail.
	body, _ := json.Marshal(*v)
	fields[key] = body
}

// sorted returns a tone entry's keys in order, so output does not depend on
// map iteration.
func sorted(entry preset.Tone) []string {
	out := make([]string, 0, len(entry))
	for k := range entry {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

// sortedKeys returns a raw map's keys in order.
func sortedKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}
