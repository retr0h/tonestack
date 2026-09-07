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

package wire

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// ErrNotAPreset reports an answer that is not a preset document.
var ErrNotAPreset = errors.New("not a preset")

// magic is what a preset from a device starts with.
const magic = "l6-helix\x00"

// Keys inside a preset document. A device names everything by number.
const (
	keyTone      = 0
	keyBlocks    = 22
	keyBlockKind = 19
	keyBlockBody = 20
	keyModelRef  = 24
	keyModelNum  = 25
	keyParams    = 11
	keyValues    = 4
	keyBypassed  = 10
)

// kindBlock marks an entry holding a model rather than routing or a gap.
const kindBlock = 6

// Keys the footswitch and snapshot sections sit under.
const (
	keyFootswitch = 3
	keyFsPaths    = 8
	keyFsBody     = 11
	keyFsModel    = 5
	keyFsBlock    = 8
	keyFsNamed    = 13
	keyFsLabel    = 14
	keyFsLED      = 16
	keySnapshots  = 10
	keySnapList   = 10
	keySnapName   = 4
	keySnapTempo  = 5
	keySnapLED    = 12
	keySnapValid  = 0
)

// DevicePreset is one preset as the hardware describes it.
type DevicePreset struct {
	// Blocks are the chain, in the order the device laid them out.
	Blocks []DeviceBlock
	// Snapshots are the block states a footswitch recalls.
	Snapshots []DeviceSnapshot
	// Footswitches are what the pedal shows on its own screen.
	Footswitches []DeviceFootswitch
}

// DeviceSnapshot is one snapshot as the hardware stores it.
type DeviceSnapshot struct {
	// Name is what the device shows. Ships as SNAPSHOT 1 through 3.
	Name string
	// Tempo is what the snapshot recalls, in beats per minute.
	Tempo float64
	// LED is the colour the footswitch lights, as the device numbers them.
	LED int
	// Valid is whether the device considers the snapshot set up.
	Valid bool
}

// DeviceFootswitch is one thing a switch on the pedal does.
type DeviceFootswitch struct {
	// Switch is which footswitch, counted the way the pedal prints them:
	// FS1 is 1.
	//
	// A switch can carry more than one assignment, so this is not unique.
	Switch int
	// Label is what the pedal prints under the switch.
	//
	// Somebody's own words when they set one, and the block's name when they
	// did not — which is what the pedal shows either way.
	Label string
	// Gear is the block's own name, whatever the label says.
	Gear string
	// Block is which block the switch works on, as the device numbers them.
	Block int
	// LED is the colour somebody chose for the switch, as the device
	// numbers its own list. Zero is "Auto Color", where the light follows
	// the block rather than a choice.
	LED int
}

// DeviceBlock is one block, still named by number.
type DeviceBlock struct {
	// Model is the block's position in the device's own model table, which
	// the catalog carries as its symbol list.
	Model int
	// Values are the block's parameters, in the order that table gives their
	// names. Position is the only thing identifying them here.
	//
	// Each is a bool, an int64 or a float64 — narrowed here, where the wire
	// format is known, so nothing downstream has to know which of MessagePack's
	// integer widths a device happened to use.
	Values []any
	// Enabled is whether the block is switched on.
	Enabled bool
}

// DecodePreset reads what a device hands back for one slot.
//
// A preset arrives as three concatenated MessagePack values: the magic string
// `l6-helix`, a table of byte offsets the device seeks with, and the preset
// itself. Only the third is read here — the offsets exist for writing, which
// this does not do.
func DecodePreset(body []byte) (DevicePreset, error) {
	dec := msgpack.NewDecoder(bytes.NewReader(body))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		return d.DecodeUntypedMap()
	})

	head, err := dec.DecodeString()
	if err != nil || head != magic {
		return DevicePreset{}, fmt.Errorf("%w: no %q header", ErrNotAPreset, "l6-helix")
	}

	// The offset table. Skipped rather than read: a device seeks with it and
	// a reader walks the MessagePack instead.
	if _, err := dec.DecodeString(); err != nil {
		return DevicePreset{}, fmt.Errorf("%w: no offset table", ErrNotAPreset)
	}

	raw, err := dec.DecodeInterface()
	if err != nil {
		return DevicePreset{}, fmt.Errorf("%w: %w", ErrNotAPreset, err)
	}

	doc, ok := raw.(map[any]any)
	if !ok {
		return DevicePreset{}, fmt.Errorf("%w: expected a map, got %T", ErrNotAPreset, raw)
	}

	return DevicePreset{
		Blocks:       blocksOf(doc),
		Snapshots:    snapshotsOf(doc),
		Footswitches: footswitchesOf(doc),
	}, nil
}

// snapshotsOf reads the snapshots a preset carries.
func snapshotsOf(doc map[any]any) []DeviceSnapshot {
	section, ok := doc[int8(keySnapshots)].(map[any]any)
	if !ok {
		return nil
	}

	entries, ok := section[int8(keySnapList)].([]any)
	if !ok {
		return nil
	}

	out := make([]DeviceSnapshot, 0, len(entries))

	for _, e := range entries {
		entry, ok := e.(map[any]any)
		if !ok {
			continue
		}

		snap := DeviceSnapshot{}
		snap.Name, _ = asString(entry[int8(keySnapName)])

		if n, ok := asFloat(entry[int8(keySnapTempo)]); ok {
			snap.Tempo = n
		}

		if n, ok := asUint(entry[int8(keySnapLED)]); ok {
			snap.LED = int(n)
		}

		snap.Valid, _ = entry[int8(keySnapValid)].(bool)

		out = append(out, snap)
	}

	return out
}

// footswitchesOf reads what the pedal shows under each switch.
//
// A label and a colour are somebody's decisions about their own pedal, and
// nothing else in a preset records them.
func footswitchesOf(doc map[any]any) []DeviceFootswitch {
	section, ok := doc[int8(keyFootswitch)].(map[any]any)
	if !ok {
		return nil
	}

	paths, ok := section[int8(keyFsPaths)].([]any)
	if !ok {
		return nil
	}

	out := []DeviceFootswitch(nil)

	// The outer list is the switches themselves, in order, and it covers more
	// of them than any one device has: an HX Stomp answers with five groups
	// and has three switches. A switch with nothing on it answers with
	// nothing, which is why the index rather than the count is what names it.
	for i, p := range paths {
		entries, ok := p.([]any)
		if !ok {
			continue
		}

		for _, e := range entries {
			entry, ok := e.(map[any]any)
			if !ok {
				continue
			}

			body, ok := entry[int8(keyFsBody)].(map[any]any)
			if !ok {
				continue
			}

			gear, ok := asString(body[int8(keyFsModel)])
			if !ok || gear == "" {
				continue
			}

			// The chosen colour sits beside the body rather than in it. The
			// body carries the block's own colour, which is the same for
			// every block of that kind and is not a choice.
			led, _ := asUint(entry[int8(keyFsLED)])
			block, _ := asUint(body[int8(keyFsBlock)])

			out = append(out, DeviceFootswitch{
				Switch: i + 1,
				Label:  labelOf(entry, gear),
				Gear:   gear,
				Block:  int(block),
				LED:    int(led),
			})
		}
	}

	return out
}

// blocksOf reads the chain out of a preset document.
//
// Entries that hold no model are the device's own: an input, an output, a
// gap where nothing is placed. They are skipped rather than reported, because
// a chain is what somebody put there.
func blocksOf(doc map[any]any) []DeviceBlock {
	tone, ok := doc[int8(keyTone)].(map[any]any)
	if !ok {
		return nil
	}

	entries, ok := tone[int8(keyBlocks)].([]any)
	if !ok {
		return nil
	}

	out := make([]DeviceBlock, 0, len(entries))

	for _, e := range entries {
		entry, ok := e.(map[any]any)
		if !ok {
			continue
		}

		if kind, ok := asUint(entry[int8(keyBlockKind)]); !ok || kind != kindBlock {
			continue
		}

		body, ok := entry[int8(keyBlockBody)].(map[any]any)
		if !ok {
			continue
		}

		block, ok := blockOf(body)
		if !ok {
			continue
		}

		out = append(out, block)
	}

	return out
}

// blockOf reads one block's model and parameters.
func blockOf(body map[any]any) (DeviceBlock, bool) {
	ref, ok := body[int8(keyModelRef)].(map[any]any)
	if !ok {
		return DeviceBlock{}, false
	}

	model, ok := asUint(ref[int8(keyModelNum)])
	if !ok {
		return DeviceBlock{}, false
	}

	out := DeviceBlock{Model: int(model), Enabled: true}

	if on, ok := body[int8(keyBypassed)].(bool); ok {
		out.Enabled = on
	}

	if params, ok := body[int8(keyParams)].(map[any]any); ok {
		if values, ok := params[int8(keyValues)].([]any); ok {
			out.Values = narrow(values)
		}
	}

	return out, true
}

// narrow reduces a device's values to the three kinds it means.
//
// MessagePack carries an integer in whichever width holds it, so the same
// parameter arrives as an int8 in one preset and an int64 in another. A
// device mixes numbers, switches and enumerated positions in one array, and
// those three distinctions are the ones that matter: narrowing them all to
// numbers turns every switch off.
func narrow(values []any) []any {
	out := make([]any, 0, len(values))

	for _, v := range values {
		switch t := v.(type) {
		case bool:
			out = append(out, t)
		case float32:
			out = append(out, float64(t))
		case float64:
			out = append(out, t)
		default:
			out = append(out, whole(v))
		}
	}

	return out
}

// whole renders an integer of any width, or keeps what is not one.
func whole(v any) any {
	// asInt already falls back to the unsigned widths.
	if n, ok := asInt(v); ok {
		return n
	}

	return v
}

// labelOf is what the pedal prints under a switch.
//
// Somebody's own words when they set them, and the block's name when they did
// not. A device carries both and flags which it is showing, because the two
// are different things: "60s / 70s" is what a player reads on stage and
// "Ampeg B-15NF" is what the block happens to be.
func labelOf(entry map[any]any, gear string) string {
	named, _ := entry[int8(keyFsNamed)].(bool)
	if !named {
		return gear
	}

	label, ok := asString(entry[int8(keyFsLabel)])
	if !ok || label == "" {
		return gear
	}

	return label
}
