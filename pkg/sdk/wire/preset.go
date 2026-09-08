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
	// keyPairedCab is the cabinet an amp carries with it. A device stores the
	// two as one block; a preset stores them as a block and a sibling.
	keyPairedCab = 12
	// keyNamedCount is how many of the values a model has names for. A paired
	// cabinet sends one more than that, which is the microphone.
	keyNamedCount = 3
)

// What a chain entry is. The device lays routing out in the same array as the
// blocks, so a chain and what wraps it arrive together.
const (
	// kindInput is where a signal arrives.
	kindInput = 0
	// kindOutput is the main pair out.
	kindOutput = 1
	// kindSplit carries the second input and the split that follows it.
	kindSplit = 2
	// kindJoin carries the second output and the join that precedes it.
	kindJoin = 3
	// kindBlock is a block somebody placed. Every other kind not named here
	// is a gap in the layout where nothing sits.
	kindBlock = 6
)

// Keys inside the routing entries.
const (
	keyInputSelect  = 5
	keyOutputSelect = 6
	keyFlowParams   = 7
	keySplitInput   = 14
	keySplitBlock   = 15
	keyJoinOutput   = 16
	keyJoinBlock    = 17
	keyFlowPosition = 13
	keyFlowEnabled  = 10
	keyFlowModel    = 8
)

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

// Keys a controller assignment sits under.
//
// Section 4 is an array of ten, one per controller, each holding the
// assignments made to it. Established against slot 27B of an HX Stomp read
// side by side with the same slot exported from HX Edit, which describes the
// one assignment it carries as
//
//	"controller": {"dsp0": {"block0": {"Pedal": {
//	  "@min": 0, "@max": 1, "@controller": 2, "@snapshot_disable": false}}}}
//
// and the device sends as
//
//	[2][0] = {0: 0, 1: {0: 2, 1: 4, 2: 0, 3: 1, 4: 0, 5: 2,
//	                    6: {28: 0, 29: 0, 41: false}, 7: 0}}
//
// Five values line up with five fields the file names, which is what makes
// them more than a guess. Key 5 repeats the controller the array is already
// indexed by and is not read. Keys 1, 4 and 7, and 28 and 29 inside key 6,
// are not understood.
const (
	// keyCtrlParam is which parameter, by its place in the model's own
	// parameter order.
	keyCtrlParam = 0
	// keyCtrlBody holds everything else about the assignment.
	keyCtrlBody = 1
	// keyCtrlBlock is the block, by the position the device lays it out at.
	keyCtrlBlock = 0
	// keyCtrlMin and keyCtrlMax are the ends of the controller's travel.
	keyCtrlMin = 2
	keyCtrlMax = 3
	// keyCtrlFlags holds the switches, of which one is understood.
	keyCtrlFlags = 6
	// keyCtrlNoSnapshot is whether snapshots leave this assignment alone.
	keyCtrlNoSnapshot = 41
	// keyControllers is the section itself.
	keyControllers = 4
)

// DevicePreset is one preset as the hardware describes it.
type DevicePreset struct {
	// Blocks are the chain, in the order the device laid them out.
	Blocks []DeviceBlock
	// Snapshots are the block states a footswitch recalls.
	Snapshots []DeviceSnapshot
	// Footswitches are what the pedal shows on its own screen.
	Footswitches []DeviceFootswitch
	// Routing is what the device wraps the chain in: its inputs, its outputs,
	// and the split and join of a parallel path.
	//
	// Read from the same array as the blocks, because that is where the
	// device puts them.
	Routing []DeviceRouting
	// Controllers are the parameters something moves: an expression pedal,
	// or a footswitch set to sweep rather than to toggle.
	Controllers []DeviceController
}

// DeviceController is one parameter something moves.
type DeviceController struct {
	// Controller is which one, as the device numbers them. The expression
	// pedal on the captured preset is 2.
	Controller int
	// Block is the block it works on, by the position the device lays it out
	// at, which is what a footswitch assignment uses too.
	Block int
	// Param is which parameter, by its place in the model's own parameter
	// order — the same order the block's values arrive in.
	Param int
	// Min and Max are the ends of its travel.
	Min float64
	Max float64
	// NoSnapshot is whether snapshots leave this assignment alone.
	NoSnapshot bool
}

// DeviceRouting is one entry either side of a chain.
type DeviceRouting struct {
	// Slot names which entry this is, as a preset names it: `inputA`,
	// `outputB`, `split`, `join`.
	Slot string
	// Model is the block's position in the device's own model table, for the
	// split and the join. An input or an output carries none, because a
	// device knows which are its own.
	Model int
	// HasModel says whether Model means anything.
	HasModel bool
	// Select is which input or output this is, as the preset records it.
	Select int
	// HasSelect says whether Select means anything.
	HasSelect bool
	// Position is where the split or join sits in the layout.
	Position int
	// Enabled is whether a split or join is switched on.
	Enabled bool
	// Values are the parameters, in the order the model table names them.
	Values []any
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
	// Index is where the block sits in the device's own layout.
	//
	// Not its place in the chain: a device lays blocks out on a fixed grid
	// and leaves gaps in it, so a chain of four can sit at 5, 6, 8 and 13.
	// Footswitch assignments address blocks by this number, so renumbering
	// them would break the only link between the two.
	Index int
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
	// Cab is the cabinet an amp carries with it, when it has one.
	//
	// A device stores an amp and its cabinet as one block. A preset stores
	// them as a block and a sibling entry, so this has to come out.
	Cab []any
	// CabNamed is how many of those values the cabinet model has names for.
	// Anything past it is the microphone.
	CabNamed int
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
		Routing:      routingOf(doc),
		Controllers:  controllersOf(doc),
	}, nil
}

// controllersOf reads the parameters something moves.
//
// The section is an array of ten, one per controller, and the index is the
// controller. Most are nil on any real preset: one expression pedal assigned
// to one parameter leaves the other nine holding nothing.
func controllersOf(doc map[any]any) []DeviceController {
	section, ok := doc[int8(keyControllers)].([]any)
	if !ok {
		return nil
	}

	out := []DeviceController(nil)

	for number, row := range section {
		made, ok := row.([]any)
		if !ok {
			continue
		}

		for _, one := range made {
			if got, ok := controllerOf(number, one); ok {
				out = append(out, got)
			}
		}
	}

	return out
}

// mustFloat reads one end of a controller's travel, which a device writes as
// a fraction or, when it is whole, as a whole number.
func mustFloat(v any) float64 {
	f, _ := asFloat(v)

	return f
}

// controllerOf reads one assignment.
func controllerOf(number int, entry any) (DeviceController, bool) {
	fields, ok := entry.(map[any]any)
	if !ok {
		return DeviceController{}, false
	}

	param, ok := asUint(fields[int8(keyCtrlParam)])
	if !ok {
		return DeviceController{}, false
	}

	body, ok := fields[int8(keyCtrlBody)].(map[any]any)
	if !ok {
		return DeviceController{}, false
	}

	block, ok := asUint(body[int8(keyCtrlBlock)])
	if !ok {
		return DeviceController{}, false
	}

	out := DeviceController{
		Controller: number,
		Block:      int(block),
		Param:      int(param),
		// asFloat reaches every integer width too, and a device writes a
		// whole end of the travel as a whole number: 1.0 arrives as 1.
		Min: mustFloat(body[int8(keyCtrlMin)]),
		Max: mustFloat(body[int8(keyCtrlMax)]),
	}

	if flags, ok := body[int8(keyCtrlFlags)].(map[any]any); ok {
		out.NoSnapshot, _ = flags[int8(keyCtrlNoSnapshot)].(bool)
	}

	return out, true
}

// routingOf reads what the device wraps the chain in.
//
// The same array the blocks come from. An entry that is not a block is one of
// four things, and each names itself the way a preset does.
func routingOf(doc map[any]any) []DeviceRouting {
	entries := chainEntries(doc)
	out := []DeviceRouting(nil)

	for _, e := range entries {
		entry, ok := e.(map[any]any)
		if !ok {
			continue
		}

		kind, ok := asUint(entry[int8(keyBlockKind)])
		if !ok {
			continue
		}

		body, _ := entry[int8(keyBlockBody)].(map[any]any)
		if body == nil {
			continue
		}

		switch kind {
		case kindInput:
			out = append(out, flowOf("inputA", body, keyInputSelect))
		case kindOutput:
			out = append(out, flowOf("outputA", body, keyOutputSelect))
		case kindSplit:
			out = appendPair(out, body,
				"inputB", keySplitInput, keyInputSelect,
				"split", keySplitBlock)
		case kindJoin:
			out = appendPair(out, body,
				"outputB", keyJoinOutput, keyOutputSelect,
				"join", keyJoinBlock)
		}
	}

	return out
}

// appendPair reads the two entries a split or a join arrives with.
//
// A device pairs its second input with the split that follows it, and its
// second output with the join that precedes it. A preset stores them apart.
func appendPair(
	out []DeviceRouting,
	body map[any]any,
	endSlot string, endKey, selectKey int,
	blockSlot string, blockKey int,
) []DeviceRouting {
	if end, ok := body[int8(endKey)].(map[any]any); ok {
		out = append(out, flowOf(endSlot, end, selectKey))
	}

	block, ok := body[int8(blockKey)].(map[any]any)
	if !ok {
		return out
	}

	got := DeviceRouting{Slot: blockSlot, Values: flowValues(block)}

	if n, ok := asUint(block[int8(keyFlowModel)]); ok {
		got.Model, got.HasModel = int(n), true
	}

	if n, ok := asUint(block[int8(keyFlowPosition)]); ok {
		got.Position = int(n)
	}

	got.Enabled, _ = block[int8(keyFlowEnabled)].(bool)

	return append(out, got)
}

// flowOf reads one input or output.
func flowOf(slot string, body map[any]any, selectKey int) DeviceRouting {
	out := DeviceRouting{Slot: slot, Values: flowValues(body)}

	if n, ok := asUint(body[int8(selectKey)]); ok {
		out.Select, out.HasSelect = int(n), true
	}

	return out
}

// flowValues reads a routing entry's parameters.
func flowValues(body map[any]any) []any {
	params, ok := body[int8(keyFlowParams)].(map[any]any)
	if !ok {
		return nil
	}

	values, ok := params[int8(keyValues)].([]any)
	if !ok {
		return nil
	}

	return narrow(values)
}

// chainEntries returns the array holding the chain and its routing.
func chainEntries(doc map[any]any) []any {
	tone, ok := doc[int8(keyTone)].(map[any]any)
	if !ok {
		return nil
	}

	entries, _ := tone[int8(keyBlocks)].([]any)

	return entries
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
	entries := chainEntries(doc)

	out := make([]DeviceBlock, 0, len(entries))

	for i, e := range entries {
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

		block.Index = i

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

	if cab, ok := body[int8(keyPairedCab)].(map[any]any); ok {
		if values, ok := cab[int8(keyValues)].([]any); ok && len(values) > 0 {
			out.Cab = narrow(values)

			if n, ok := asUint(cab[int8(keyNamedCount)]); ok {
				out.CabNamed = int(n)
			}
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

// Loaded is which preset a device is playing.
type Loaded struct {
	// Setlist is which setlist it came from.
	Setlist int
	// Slot is its position within that setlist, counted from zero.
	Slot int
	// Name is what the device calls it.
	Name string
}

// DecodeLoaded reads what a device answers when asked what it is playing.
//
// The one honest signal that a select has finished. A device takes a select
// and completes it afterwards, and answers other questions while the switch
// is still in flight, so "it answered again" is not "it finished".
func DecodeLoaded(result any) (Loaded, error) {
	body, ok := result.(map[any]any)
	if !ok {
		return Loaded{}, fmt.Errorf(
			"%w: expected a map, got %T", ErrNotAPreset, result)
	}

	setlist, ok := asUint(body[int8(keySetlist)])
	if !ok {
		return Loaded{}, fmt.Errorf("%w: it names no setlist", ErrNotAPreset)
	}

	slot, ok := asUint(body[int8(keyPresetIndex)])
	if !ok {
		return Loaded{}, fmt.Errorf("%w: it names no slot", ErrNotAPreset)
	}

	name, _ := asString(body[int8(keyPresetName)])

	return Loaded{Setlist: int(setlist), Slot: int(slot), Name: name}, nil
}
