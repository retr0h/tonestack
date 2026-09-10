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
	"slices"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
)

// Read decodes a preset file.
func Read(r io.Reader) (*Document, error) {
	var doc Document
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decoding preset: %w", err)
	}

	if doc.Schema != Schema {
		return nil, &NotAPresetError{Schema: doc.Schema}
	}

	return &doc, nil
}

// Spec extracts the signal chain a document describes.
//
// Only processor entries are read. Structural entries — inputs, outputs,
// splits, joins — describe routing rather than a chain and are left in the
// document, which is why Write needs the document it came from to reproduce
// a preset faithfully.
func (d *Document) Spec() (chain.Chain, error) { return d.Data.Spec() }

// Spec extracts the signal chain a payload describes.
//
// This is the level a setlist addresses. A slot in a backup holds a Data and
// nothing around it, so the conversion belongs here and Document delegates.
func (d *Data) Spec() (chain.Chain, error) {
	spec := chain.Chain{Name: d.Meta.Name}

	for _, key := range sortedProcessors(d.Tone) {
		dsp, err := processorIndex(key)
		if err != nil {
			return chain.Chain{}, err
		}

		blocks, err := readBlocks(d.Tone[key])
		if err != nil {
			return chain.Chain{}, fmt.Errorf("%s: %w", key, err)
		}

		for _, b := range blocks {
			spec.Blocks = append(spec.Blocks, chain.Block{
				Model:   b.Model,
				Params:  b.Params,
				Attrs:   b.Attrs,
				DSP:     dsp,
				Pos:     b.Slot,
				Enabled: b.Enabled,
			})
		}
	}

	return spec, nil
}

// keep records an attribute exactly as it arrived.
func keep(b *block, key string, val json.RawMessage) {
	if b.Attrs == nil {
		b.Attrs = map[string]json.RawMessage{}
	}

	b.Attrs[key] = val
}

// assignTyped fills the attributes this package models as fields.
func assignTyped(b *block, key string, val json.RawMessage) error {
	switch key {
	case attrPath:
		return json.Unmarshal(val, &b.Path)
	case attrStereo:
		return json.Unmarshal(val, &b.Stereo)
	default:
		return json.Unmarshal(val, &b.Type)
	}
}

// sortedProcessors returns the dspN keys in index order, so a chain is read
// the same way every time.
func sortedProcessors(t map[string]Tone) []string {
	var keys []string

	for k := range t {
		if strings.HasPrefix(k, "dsp") {
			keys = append(keys, k)
		}
	}

	// By the number rather than by the name: sorting the strings puts dsp10
	// ahead of dsp2, which no device has yet and the comment above promises
	// not to do.
	slices.SortFunc(keys, func(a, b string) int {
		x, errA := processorIndex(a)
		y, errB := processorIndex(b)

		if errA != nil || errB != nil {
			return strings.Compare(a, b)
		}

		return x - y
	})

	return keys
}

// processorIndex reads the number out of a dspN key.
func processorIndex(key string) (int, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(key, "dsp"))
	if err != nil {
		return 0, fmt.Errorf("unreadable processor key %q: %w", key, err)
	}

	return n, nil
}

// readBlocks decodes the blockN entries of one processor, in position order.
//
// Entries that are not blocks — cab0, inputA, split, join — are skipped: they
// describe routing, not a link in the chain.
func readBlocks(t Tone) ([]block, error) {
	var out []block

	for key, raw := range t {
		if !strings.HasPrefix(key, "block") {
			continue
		}

		b, err := decodeBlock(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}

		if b.Model == "" {
			continue
		}

		if b.Slot, err = strconv.Atoi(strings.TrimPrefix(key, "block")); err != nil {
			return nil, fmt.Errorf("%s: block key is not numbered: %w", key, err)
		}

		out = append(out, b)
	}

	// By position, then by the key the preset gave it. Two blocks can share a
	// position — a hand-written preset that states none puts them all at
	// zero — and an order that depends on which way a map ranged is one this
	// package's round-trip guarantee cannot hold.
	slices.SortFunc(out, func(a, b block) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}

		return a.Slot - b.Slot
	})

	return out, nil
}

// decodeBlock splits one block object into attributes and parameters.
func decodeBlock(raw json.RawMessage) (block, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return block{}, fmt.Errorf("decoding block: %w", err)
	}

	b := block{Params: make(map[string]catalog.ParamValue)}

	for key, val := range fields {
		if err := assign(&b, key, val); err != nil {
			return block{}, err
		}
	}

	return b, nil
}

// assign puts one field of a block object where it belongs.
func assign(b *block, key string, val json.RawMessage) error {
	switch key {
	case attrModel:
		return json.Unmarshal(val, &b.Model)
	case attrPosition:
		// Kept raw as well as typed. A block's key and its position are
		// independent — block5 can carry @position 6 — so the attribute
		// travels rather than being derived from the key.
		keep(b, key, val)

		return json.Unmarshal(val, &b.Position)
	case attrEnabled:
		return json.Unmarshal(val, &b.Enabled)
	case attrPath, attrStereo, attrType:
		// Typed for convenience and kept raw as well. A chain has no opinion
		// about which path a block sits on, so it carries the attribute
		// rather than deciding it, and a preset written back has the value
		// the device put there.
		keep(b, key, val)

		return assignTyped(b, key, val)
	}

	if strings.HasPrefix(key, "@") {
		// An attribute this package does not model. Kept verbatim rather than
		// dropped: it is not a parameter, and rewriting a preset without it
		// changes a file nobody asked to change.
		keep(b, key, val)

		return nil
	}

	var pv catalog.ParamValue
	if err := pv.UnmarshalJSON(val); err != nil {
		return fmt.Errorf("parameter %q: %w", key, err)
	}

	b.Params[key] = pv

	return nil
}
