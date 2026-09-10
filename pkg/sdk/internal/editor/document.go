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
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	riggen "github.com/retr0h/tonestack/pkg/sdk/internal/gen"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// Controllers carries what an expression pedal or a footswitch moves.
//
// A device stores the parameter as a number, its place in the model's own
// order, and stores the block by the position it lays it out at. Neither
// reads, so the catalog turns the first into a name and the grid offset turns
// the second into a place along the path.
//
// An assignment naming a model or a parameter this catalog cannot reach is
// dropped rather than written with a number in place of a name. A rig that
// said `parameter: 4` would be unreadable and would mean something different
// after the next firmware release.
func Controllers(
	got wire.DevicePreset,
	cat *catalog.Catalog,
) *[]riggen.Controller {
	if len(got.Controllers) == 0 {
		return nil
	}

	out := make([]riggen.Controller, 0, len(got.Controllers))

	for _, c := range got.Controllers {
		name, ok := paramNameOf(got, cat, c.Block, c.Param)
		if !ok {
			continue
		}

		lo, hi := float32(c.Min), float32(c.Max)
		one := riggen.Controller{
			Controller: c.Controller,
			Block:      c.Block - wire.GridOffset,
			Parameter:  name,
			Min:        &lo,
			Max:        &hi,
		}

		if c.NoSnapshot {
			one.NoSnapshot = &c.NoSnapshot
		}

		out = append(out, one)
	}

	if len(out) == 0 {
		return nil
	}

	return &out
}

// paramNameOf names one parameter of one block.
//
// The block is found by the position the device laid it out at, because that
// is how a controller assignment addresses it.
func paramNameOf(
	got wire.DevicePreset,
	cat *catalog.Catalog,
	at, param int,
) (string, bool) {
	for _, b := range got.Blocks {
		if b.Index != at {
			continue
		}

		sym, ok := cat.Symbol(b.Model)
		if !ok || param < 0 || param >= len(sym.Params) {
			return "", false
		}

		return sym.Params[param], true
	}

	return "", false
}

// Document builds the preset a device's answer describes.
//
// Everything a preset holds, not only the chain: the routing a device wraps
// one in, and the cabinets its amplifiers carry, which a preset keeps as
// sibling entries rather than inside the block.
//
// The blank it is written into is an untouched preset the device itself
// wrote, embedded in this binary and read by a test, so it cannot fail to
// decode.
func Document(
	got wire.DevicePreset,
	cat *catalog.Catalog,
	name string,
) (*preset.Document, bool, error) {
	c, err := Chain(name, got, cat)
	if err != nil {
		return nil, false, err
	}

	if len(c.Blocks) == 0 {
		return nil, true, nil
	}

	doc, _ := preset.Blank()
	doc.Data.Device = cat.DeviceID
	doc.Data.Meta.Name = name

	_ = doc.SetSpec(c)

	if state := routingOf(got, cat); state != nil {
		for key, body := range *state {
			doc.Data.Tone[processorKey][strings.TrimPrefix(
				key, processorKey+".")] = body
		}
	}

	return doc, false, nil
}

// Snapshots carries what the device recalls on a footswitch.
func Snapshots(got wire.DevicePreset) *[]riggen.Snapshot {
	if len(got.Snapshots) == 0 {
		return nil
	}

	out := make([]riggen.Snapshot, 0, len(got.Snapshots))

	for _, s := range got.Snapshots {
		snap := riggen.Snapshot{}

		if s.Name != "" {
			name := s.Name
			snap.Name = &name
		}

		if s.Tempo > 0 {
			tempo := s.Tempo
			snap.Tempo = &tempo
		}

		led, valid := s.LED, s.Valid
		snap.Led, snap.Valid = &led, &valid

		out = append(out, snap)
	}

	return &out
}

// Footswitches carries what the pedal prints under each switch.
func Footswitches(got wire.DevicePreset, cat *catalog.Catalog) *[]riggen.Footswitch {
	if len(got.Footswitches) == 0 {
		return nil
	}

	out := make([]riggen.Footswitch, 0, len(got.Footswitches))

	for _, f := range got.Footswitches {
		// The block as the chain numbers it, so a footswitch and the entry
		// it works on agree.
		label, gear, at, block := f.Label, f.Gear, f.Switch, f.Block-wire.GridOffset
		fs := riggen.Footswitch{
			Switch: &at, Label: &label, Gear: &gear, Block: &block,
		}

		// A colour a device knows and this catalog does not gets no name
		// rather than a wrong one.
		if name, ok := cat.LEDColour(f.LED); ok {
			fs.Led = &name
		}

		out = append(out, fs)
	}

	return &out
}
