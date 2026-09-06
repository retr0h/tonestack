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
	"sort"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/rig"
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
func (d *Document) Spec() (rig.Spec, error) {
	spec := rig.Spec{Name: d.Data.Meta.Name, Origin: rig.OriginCurated}

	for _, key := range sortedProcessors(d.Data.Tone) {
		dsp, err := processorIndex(key)
		if err != nil {
			return rig.Spec{}, err
		}

		blocks, err := readBlocks(d.Data.Tone[key])
		if err != nil {
			return rig.Spec{}, fmt.Errorf("%s: %w", key, err)
		}

		for _, b := range blocks {
			spec.Blocks = append(spec.Blocks, rig.SpecBlock{
				Model:   b.Model,
				Params:  b.Params,
				DSP:     dsp,
				Pos:     b.Position,
				Enabled: b.Enabled,
			})
		}
	}

	return spec, nil
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

	sort.Strings(keys)

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
func readBlocks(t Tone) ([]Block, error) {
	var out []Block

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

		out = append(out, b)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })

	return out, nil
}

// decodeBlock splits one block object into attributes and parameters.
func decodeBlock(raw json.RawMessage) (Block, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Block{}, fmt.Errorf("decoding block: %w", err)
	}

	b := Block{Params: make(map[string]catalog.ParamValue)}

	for key, val := range fields {
		if err := assign(&b, key, val); err != nil {
			return Block{}, err
		}
	}

	return b, nil
}

// assign puts one field of a block object where it belongs.
func assign(b *Block, key string, val json.RawMessage) error {
	switch key {
	case attrModel:
		return json.Unmarshal(val, &b.Model)
	case attrPosition:
		return json.Unmarshal(val, &b.Position)
	case attrEnabled:
		return json.Unmarshal(val, &b.Enabled)
	case attrPath:
		return json.Unmarshal(val, &b.Path)
	case attrStereo:
		return json.Unmarshal(val, &b.Stereo)
	case attrType:
		return json.Unmarshal(val, &b.Type)
	}

	if strings.HasPrefix(key, "@") {
		// An attribute this package does not model. Not a parameter, so
		// leaving it out of Params keeps a value the device owns from being
		// written back as if we chose it.
		return nil
	}

	var pv catalog.ParamValue
	if err := pv.UnmarshalJSON(val); err != nil {
		return fmt.Errorf("parameter %q: %w", key, err)
	}

	b.Params[key] = pv

	return nil
}
