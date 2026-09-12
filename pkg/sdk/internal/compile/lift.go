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
	"fmt"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Lift reads a preset into a rig.
//
// Every block records both the gear it emulates and the exact model it was,
// keyed by device. The name alone cannot identify a model — 665 of them share
// 469 names — so a rig that only carried the name would rebuild into a
// different preset.
func Lift(doc *preset.Document, cat *catalog.Catalog) (rig.Spec, error) {
	c, err := doc.Spec()
	if err != nil {
		return rig.Spec{}, fmt.Errorf("reading the chain: %w", err)
	}

	device := cat.Device
	// The contract states one version and the generated types carry it as a
	// kind of its own, so this is a conversion rather than a number.
	version := rig.SpecVersion(rig.Version)

	entries := make([]rig.ChainEntry, 0, len(c.Blocks))

	for _, b := range c.Blocks {
		entries = append(entries, entryFor(b, cat, device))
	}

	snapshots := snapshotsOf(doc)
	switches := footswitchesOf(doc)

	out := rig.Spec{
		Schema:       rig.SchemaName,
		Version:      &version,
		ID:           identifier(doc.Data.Meta.Name),
		Subject:      rig.Subject{Kind: rig.KindSound, Name: subjectName(doc)},
		Chain:        entries,
		Instrument:   instrumentOf(c, cat),
		Target:       &rig.Target{Device: &device},
		Snapshots:    snapshots,
		Footswitches: switches,
		Device:       deviceState(doc, modelledKeys(doc, snapshots, switches)),
	}

	// A rig this package produced must be one anybody else can read. Lifting
	// something that does not meet its own contract is a bug here, not input
	// worth passing on.
	if err := rig.Validate(out); err != nil {
		return rig.Spec{}, fmt.Errorf("lifting %q: %w", doc.Data.Meta.Name, err)
	}

	return out, nil
}

// subjectName is what the rig is called.
//
// A preset carries a name and nothing about who plays it, so a lifted rig is
// a sound rather than an artist until somebody says otherwise.
func subjectName(doc *preset.Document) string {
	if name := strings.TrimSpace(doc.Data.Meta.Name); name != "" {
		return name
	}

	return "Untitled"
}

// identifier turns a preset name into the shape the schema states for one.
func identifier(name string) string {
	var b strings.Builder

	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}

	id := strings.Trim(collapse(b.String()), "-")
	if id == "" {
		return "untitled"
	}

	return id
}

// collapse reduces runs of hyphens to one, which the pattern requires.
func collapse(s string) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	return s
}

// entryFor describes one block as gear.
func entryFor(b chain.Block, cat *catalog.Catalog, device string) rig.ChainEntry {
	blk, known := cat.Block(b.Model)

	pos, path := b.Pos, b.DSP

	entry := rig.ChainEntry{
		Gear:     gearName(blk, b.Model, known),
		Enabled:  &b.Enabled,
		Models:   &map[string]string{device: string(b.Model)},
		Position: &pos,
		Path:     &path,
	}

	entry.Role = roleFor(blk.Category, known)

	params := map[string]any{}

	for key, v := range b.Params {
		params[key] = v
	}

	// Attributes the device owns travel alongside the parameters. They are
	// @-prefixed, so nothing can collide, and a rig that dropped them would
	// rebuild into a preset that differs from the one it was read from.
	for key, raw := range b.Attrs {
		params[key] = json.RawMessage(raw)
	}

	if len(params) > 0 {
		entry.Params = &params
	}

	return entry
}

// gearName describes a block the way a person would.
//
// The gear it emulates when Line 6 say what that is, and the model's own name
// otherwise — every Line 6 original reads "Line 6 Original", which names
// nothing.
func gearName(blk catalog.Block, id catalog.ModelID, known bool) string {
	if !known {
		return string(id)
	}

	if blk.BasedOn != "" && blk.BasedOn != "Line 6 Original" {
		return blk.BasedOn
	}

	if blk.Name != "" {
		return blk.Name
	}

	return string(id)
}

// roleFor maps a catalog category onto the rig vocabulary.
//
// A model the catalog has never heard of is described as other rather than
// left blank: a rig has to say what every block is, and "something this
// device carries and we do not recognise" is a truthful answer.
func roleFor(c catalog.Category, known bool) rig.Role {
	if !known {
		return rig.RoleOther
	}

	if role, ok := roles[c]; ok {
		return role
	}

	return rig.RoleOther
}

// roles is the correspondence between what the catalog calls a block and what
// a rig calls it. They are deliberately the same words.
var roles = map[catalog.Category]rig.Role{
	catalog.CategoryAmp:     rig.RoleAmp,
	catalog.CategoryCab:     rig.RoleCab,
	catalog.CategoryDrive:   rig.RoleDrive,
	catalog.CategoryComp:    rig.RoleComp,
	catalog.CategoryGate:    rig.RoleGate,
	catalog.CategoryEQ:      rig.RoleEQ,
	catalog.CategoryMod:     rig.RoleMod,
	catalog.CategoryDelay:   rig.RoleDelay,
	catalog.CategoryReverb:  rig.RoleReverb,
	catalog.CategoryWah:     rig.RoleWah,
	catalog.CategoryPitch:   rig.RolePitch,
	catalog.CategoryFilter:  rig.RoleFilter,
	catalog.CategoryUtility: rig.RoleUtility,
	catalog.CategoryOther:   rig.RoleOther,
}

// instrumentOf reports which instrument a chain is for, from its amplifier.
//
// Line 6 tag amps Guitar or Bass. A chain with no amp names no instrument, so
// guitar stands as the more common default.
func instrumentOf(c chain.Chain, cat *catalog.Catalog) rig.Instrument {
	for _, b := range c.Blocks {
		blk, known := cat.Block(b.Model)
		if !known || blk.Category != catalog.CategoryAmp {
			continue
		}

		if blk.Subcategory == "Bass" {
			return rig.InstrumentBass
		}
	}

	return rig.InstrumentGuitar
}

// modelledKeys names the tone entries a rig carries as fields of its own.
//
// Only the ones it actually carried: a preset can hold an empty footswitch
// section, which produces no fields and is still something the file said.
func modelledKeys(
	doc *preset.Document,
	snapshots *[]rig.Snapshot,
	switches *[]rig.Footswitch,
) map[string]bool {
	out := map[string]bool{}

	if snapshots != nil {
		for key := range doc.Data.Tone {
			if snapshotIndex(key) >= 0 {
				out[key] = true
			}
		}
	}

	if switches != nil {
		out[footswitchKey] = true
	}

	return out
}
