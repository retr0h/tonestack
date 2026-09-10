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
package preset

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

// Write encodes a preset file.
func Write(w io.Writer, d *Document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(d); err != nil {
		return fmt.Errorf("encoding preset: %w", err)
	}

	return nil
}

// SetSpec replaces the document's signal chain with spec.
//
// Only processor blocks are rewritten. Routing, snapshots and controller
// assignments are left as they were, because a chain says nothing about them
// and discarding what the device wrote would produce a preset that loads
// differently for reasons nobody asked for.
func (d *Document) SetSpec(spec chain.Chain) error {
	d.Data.Meta.Name = spec.Name

	if d.Data.Tone == nil {
		d.Data.Tone = map[string]Tone{}
	}

	byProcessor := map[int][]chain.Block{}
	for _, b := range spec.Blocks {
		byProcessor[b.DSP] = append(byProcessor[b.DSP], b)
	}

	for dsp, blocks := range byProcessor {
		key := "dsp" + strconv.Itoa(dsp)

		tone := d.Data.Tone[key]
		if tone == nil {
			tone = Tone{}
		}

		// Drop the blocks that were there; keep everything else.
		for k := range tone {
			if isBlockKey(k) {
				delete(tone, k)
			}
		}

		for i, b := range blocks {
			raw, err := encodeBlock(b)
			if err != nil {
				return fmt.Errorf("%s block %d: %w", key, i, err)
			}

			tone["block"+strconv.Itoa(b.Pos)] = raw
		}

		d.Data.Tone[key] = tone
	}

	return nil
}

// isBlockKey reports whether a tone entry is a chain block rather than
// routing.
func isBlockKey(k string) bool {
	if len(k) <= len("block") || k[:len("block")] != "block" {
		return false
	}

	_, err := strconv.Atoi(k[len("block"):])

	return err == nil
}

// encodeBlock renders one block as the device writes it: @-prefixed
// attributes alongside parameters, each parameter in its own kind.
func encodeBlock(b chain.Block) (json.RawMessage, error) {
	fields := map[string]any{
		attrModel:   string(b.Model),
		attrEnabled: b.Enabled,
	}

	// Position is an attribute rather than the block's key, so it is only
	// derived from the key when a chain carried none — which is the case for
	// a chain this tool built rather than read.
	if _, ok := b.Attrs[attrPosition]; !ok {
		fields[attrPosition] = b.Pos
	}

	for k, v := range b.Params {
		if _, taken := fields[k]; taken {
			return nil, fmt.Errorf("parameter %q collides with an attribute", k)
		}

		fields[k] = v
	}

	// Attributes the chain carried but has no opinion about, put back as
	// they arrived.
	for k, v := range b.Attrs {
		if _, taken := fields[k]; taken {
			continue
		}

		fields[k] = v
	}

	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encoding block: %w", err)
	}

	return raw, nil
}

// New returns a document for a device, carrying spec.
func New(deviceID int, spec chain.Chain) (*Document, error) {
	d := &Document{
		Schema:  Schema,
		Version: Version,
		Data:    Data{Device: deviceID, Tone: map[string]Tone{}},
	}

	if err := d.SetSpec(spec); err != nil {
		return nil, err
	}

	return d, nil
}
