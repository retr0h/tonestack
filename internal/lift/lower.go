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
	"fmt"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/rig"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// Lower writes a rig back into a preset.
//
// The document is written into rather than built, because a preset holds
// things a rig does not model — the inputs, outputs, split and join a device
// expects, its snapshots, its controller assignments. Building one from
// nothing would produce a file unlike any the device has ever written.
func Lower(
	doc *preset.Document,
	spec riggen.RigSpec,
	cat *catalog.Catalog,
) error {
	// Checked on the way in as well as on the way out. A rig can arrive from
	// anywhere — a file somebody wrote, a model that generated one — and
	// building a preset out of one that does not meet its own contract turns
	// a legible error into a device refusing a file.
	if err := rig.Validate(spec); err != nil {
		return err
	}

	blocks := make([]chain.Block, 0, len(spec.Chain))

	for i, entry := range spec.Chain {
		model, err := modelFor(entry, cat)
		if err != nil {
			return fmt.Errorf("chain entry %d: %w", i, err)
		}

		blocks = append(blocks, chain.Block{
			Model:   model,
			Params:  paramsFor(entry, cat, model),
			Attrs:   attrsFor(entry),
			DSP:     at(entry.Path, 0),
			Pos:     at(entry.Position, i),
			Enabled: entry.Enabled == nil || *entry.Enabled,
		})
	}

	// Before the chain, so a rig that carries routing writes its own rather
	// than keeping whatever the preset underneath came with.
	restore(doc, spec.Device)

	// After the device's own state, because a rig's snapshots are its own
	// even when it carries a verbatim record of everything else.
	if spec.Snapshots != nil {
		pruneSnapshots(doc)
		restoreSnapshots(doc, *spec.Snapshots)
	}

	if spec.Footswitches != nil {
		pruneFootswitches(doc)
		restoreFootswitches(doc, *spec.Footswitches)
	}

	// The rig names the preset, not the document underneath: compiling into
	// an untouched preset would otherwise write out the template's own name.
	// A lifted rig carries the label the device stored, padding and all,
	// which is what restore has already put back.
	name := spec.Subject.Name
	if spec.Device != nil && spec.Device.Name != nil {
		name = *spec.Device.Name
	}

	return doc.SetSpec(chain.Chain{Name: name, Blocks: blocks})
}

// at reads an optional integer, falling back when a rig does not state one.
func at(v *int, fallback int) int {
	if v == nil {
		return fallback
	}

	return *v
}

// modelFor decides which model an entry means.
//
// An exact identifier recorded for this device wins, because it is what was
// actually there. Falling back to the gear name is right for a rig written by
// hand, and wrong for one lifted off a device: 665 models share 469 names, so
// the name alone would resolve to a different model than the one recorded.
func modelFor(
	entry riggen.ChainEntry,
	cat *catalog.Catalog,
) (catalog.ModelID, error) {
	if entry.Models != nil {
		if id, ok := (*entry.Models)[cat.Device]; ok {
			// Used whether or not the catalog carries it. A preset can name
			// a model from newer firmware than the catalog was generated
			// from, and the catalog is what this tool knows rather than a
			// statement about what the device had. Refusing here would
			// rewrite somebody's preset into a different one.
			return catalog.ModelID(id), nil
		}
	}

	for id, b := range cat.Blocks {
		if b.Matches(entry.Gear) {
			return id, nil
		}
	}

	return "", fmt.Errorf("nothing on this device is %q", entry.Gear)
}

// paramsFor decides what every knob on a block is set to.
//
// The two layers mean different things, and conflating them is what makes a
// round trip lossy.
//
// A rig carrying device parameters is describing a block exactly — it was
// lifted from a preset, or somebody dialled it. Those values are the whole
// truth, and adding catalog defaults on top would write knobs the original
// did not have.
//
// A rig carrying none is describing gear rather than a block. There the
// catalog's defaults are the answer, since Line 6 state one for every
// parameter and it is never invalid.
func paramsFor(
	entry riggen.ChainEntry,
	cat *catalog.Catalog,
	model catalog.ModelID,
) chain.Params {
	blk, known := cat.Block(model)
	out := chain.Params{}

	if entry.Params == nil {
		if known {
			for key, p := range blk.Params {
				if p.Default.Type() == "" {
					continue
				}

				out[key] = p.Default
			}
		}

		return out
	}

	for key, v := range *entry.Params {
		if strings.HasPrefix(key, "@") {
			continue
		}

		if pv, ok := paramValue(blk, key, v, known); ok {
			out[key] = pv
		}
	}

	return out
}

// attrsFor pulls the device attributes back out of a rig's parameters.
//
// They travel together because a device mixes them in one block, and they are
// told apart by the @ prefix the format itself uses.
func attrsFor(entry riggen.ChainEntry) map[string]json.RawMessage {
	if entry.Params == nil {
		return nil
	}

	out := map[string]json.RawMessage{}

	for key, v := range *entry.Params {
		if !strings.HasPrefix(key, "@") {
			continue
		}

		// The value came out of a decoded document, so it encodes again.
		raw, _ := json.Marshal(v)
		out[key] = raw
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// paramValue converts a rig's value into the kind the device accepts.
//
// A device mixes floats, integers, switches and enumerations inside one
// block, and it does not coerce between them: given 1.5 for a three-position
// switch it refuses the preset rather than rounding.
func paramValue(
	blk catalog.Block,
	key string,
	v any,
	known bool,
) (catalog.ParamValue, bool) {
	// A round trip hands the value straight back, already typed.
	if pv, ok := v.(catalog.ParamValue); ok {
		return pv, pv.Type() != ""
	}

	switch t := v.(type) {
	case bool:
		return catalog.Bool(t), true
	case string:
		return catalog.Enum(t), true
	case float64:
		if known {
			if p, ok := blk.Params[key]; ok && p.Default.Type() == catalog.ParamInt {
				return catalog.Int(int64(t + 0.5)), true
			}
		}

		return catalog.Float(t), true
	default:
		return catalog.ParamValue{}, false
	}
}
