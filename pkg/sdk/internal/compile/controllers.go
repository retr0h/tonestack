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
	"strconv"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Keys a preset stores one controller assignment under. A device owns these
// names; all 14,528 assignments in the corpus carry the first three, and
// 2,971 of them carry the fourth.
const (
	controllerKey = "controller"
	ctlNumber     = "@controller"
	ctlMin        = "@min"
	ctlMax        = "@max"
	ctlNoSnapshot = "@snapshot_disable"
)

// Controllers writes what an expression pedal or a footswitch moves.
//
// A rig records these because they are decisions somebody made about how they
// play. Nothing else says the pedal under your foot is on the amplifier's
// drive rather than its volume, and until this existed a preset built from a
// rig that said so had the pedal doing nothing.
//
// Addressed the way the preset addresses its blocks: by path and by the
// position a block is stored under, which is what the chain has just been
// written with. A rig names the block by its place along the path, so a chain
// the device split across two processors still lands on the right one.
func Controllers(
	doc *preset.Document,
	spec rig.Spec,
	blocks []chain.Block,
	cat *catalog.Catalog,
) error {
	if spec.Controllers == nil {
		return nil
	}

	// Checked before anything is written, so a rig that will not build leaves
	// the preset as it was.
	if err := checkControllers(spec, blocks, cat); err != nil {
		return err
	}

	// Replaced rather than merged. The rig's list is the whole of what it
	// says moves, so an assignment the preset underneath came with is one
	// nobody asked for.
	paths := map[string]map[string]map[string]preset.Tone{}

	for _, c := range *spec.Controllers {
		// The check above has already refused a block the chain does not
		// have, so this one is there.
		at, _ := blockAt(blocks, c.Block)

		path := processorKey(at.DSP)
		block := "block" + strconv.Itoa(at.Pos)

		if paths[path] == nil {
			paths[path] = map[string]map[string]preset.Tone{}
		}

		if paths[path][block] == nil {
			paths[path][block] = map[string]preset.Tone{}
		}

		paths[path][block][c.Parameter] = assignment(c, cat, at.Model)
	}

	entry := preset.Tone{}

	for path, blocks := range paths {
		// Every value here is a number or a boolean this package wrote, so
		// there is nothing in it that will not encode.
		body, _ := json.Marshal(blocks)
		entry[path] = body
	}

	doc.Data.Tone[controllerKey] = entry

	return nil
}

// assignment is one parameter and what the controller moves it between.
//
// A rig that names no ends gets the control's own, which is the full sweep
// and the only answer that does not invent a range somebody did not ask for.
func assignment(
	c rig.Controller,
	cat *catalog.Catalog,
	model catalog.ModelID,
) preset.Tone {
	// Kept at the width a rig states them in. Widening 0.85 to a float64
	// writes 0.8500000238418579, and a preset should read the way the person
	// wrote it.
	lo, hi := float32(0), float32(1)

	if blk, ok := cat.Block(model); ok {
		if p, ok := blk.Params[c.Parameter]; ok {
			lo, hi = float32(p.Min), float32(p.Max)
		}
	}

	if c.Min != nil {
		lo = float32(*c.Min)
	}

	if c.Max != nil {
		hi = float32(*c.Max)
	}

	out := preset.Tone{}
	put(out, ctlNumber, &c.Controller)
	put(out, ctlMin, &lo)
	put(out, ctlMax, &hi)

	// Written only when the rig says so. Two thirds of the presets in the
	// corpus that assign anything leave the field out entirely.
	if c.NoSnapshot != nil {
		put(out, ctlNoSnapshot, c.NoSnapshot)
	}

	return out
}
