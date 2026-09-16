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
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// Moving snapshots between a device and a file, in both directions.
//
// A snapshot is three sounds over one chain: which blocks are on, under what
// name, at what tempo. A device records which blocks by the position it lays
// them out at, and a preset records it by the entry each block is stored
// under. Those are not the same number, so neither direction can copy it
// across.

// Keys a preset stores a snapshot under. A device owns these names.
const (
	snapshotPrefix = "snapshot"
	snapName       = "@name"
	snapTempo      = "@tempo"
	snapLED        = "@ledcolor"
	snapValid      = "@valid"
	snapBlocks     = "blocks"
)

// blockPrefix is what a preset names a chain entry with.
const blockPrefix = "block"

// snapshotsInto writes what each of a device's snapshots recalls into a
// preset.
//
// The preset is written into an untouched one the device wrote, which ships
// three snapshots named SNAPSHOT 1 to 3 with nothing set on them. Leaving
// those is what made an exported file claim every device's snapshots were
// unused and identical.
//
// The chain has to be written first: a snapshot names the blocks it switches
// by the entry they are stored under, and this reads those entries to find
// them.
func snapshotsInto(
	doc *preset.Document,
	got wire.DevicePreset,
) {
	for i, s := range got.Snapshots {
		key := snapshotPrefix + strconv.Itoa(i)

		// Into whatever the template holds, rather than over it. A snapshot
		// carries fields nothing here reads, and a file that dropped them
		// would differ from the device's for reasons nobody asked for.
		entry := doc.Data.Tone[key]
		if entry == nil {
			entry = preset.Tone{}
		}

		fields := map[string]any{
			snapName:  s.Name,
			snapTempo: s.Tempo,
			snapLED:   s.LED,
			snapValid: s.Valid,
		}

		if blocks := snapshotBlocks(got, s); blocks != nil {
			fields[snapBlocks] = map[string]any{processorKey: blocks}
		}

		for name, v := range fields {
			// Values that came off the wire, and maps of them, always marshal.
			body, _ := json.Marshal(v)
			entry[name] = body
		}

		doc.Data.Tone[key] = entry
	}
}

// snapshotBlocks names the blocks one snapshot switches, the way a preset
// names them.
//
// By the entry a block is stored under rather than by the grid position the
// device records it at. The two differ by wire.GridOffset, and this is written
// alongside a chain that was written the same way.
func snapshotBlocks(
	got wire.DevicePreset,
	s wire.DeviceSnapshot,
) map[string]any {
	if len(s.On) == 0 {
		return nil
	}

	out := map[string]any{}

	for _, b := range got.Blocks {
		on, named := s.On[b.Index]
		if !named {
			continue
		}

		out[blockPrefix+strconv.Itoa(b.Index-wire.GridOffset)] = on
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// SnapshotStates reads what a preset's snapshots recall.
//
// The way back from snapshotsInto, for putting a file on a device. A snapshot
// names blocks by the entry they are stored under, and each name is resolved
// through that entry's own position rather than through the number in it: a
// preset can store block5 at position 6, and one that trusted the name would
// switch the wrong block.
func SnapshotStates(
	doc *preset.Document,
) []wire.Snapshot {
	keys := make([]string, 0, len(doc.Data.Tone))

	for key := range doc.Data.Tone {
		if snapshotIndex(key) >= 0 {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return nil
	}

	// In the order the device numbers them, which is the order it recalls
	// them in. Nothing else refers to a snapshot by name.
	sort.Slice(keys, func(i, j int) bool {
		return snapshotIndex(keys[i]) < snapshotIndex(keys[j])
	})

	at := positionsOf(doc)

	out := make([]wire.Snapshot, 0, len(keys))
	for _, key := range keys {
		out = append(out, snapshotStateOf(doc.Data.Tone[key], at))
	}

	return out
}

// snapshotStateOf reads one snapshot entry.
func snapshotStateOf(
	entry preset.Tone,
	at map[string]int,
) wire.Snapshot {
	var out wire.Snapshot

	readInto(entry[snapName], &out.Name)
	readInto(entry[snapTempo], &out.Tempo)
	readInto(entry[snapLED], &out.LED)
	readInto(entry[snapValid], &out.Valid)

	out.On = snapshotOn(entry[snapBlocks], at)

	return out
}

// positionsOf is where each of a preset's entries sits on the device's grid.
//
// From the entry's own position rather than the number in its name, because a
// preset is free to store block5 at position 6 and a real one does.
func positionsOf(
	doc *preset.Document,
) map[string]int {
	out := map[string]int{}

	for key, raw := range doc.Data.Tone[processorKey] {
		var fields struct {
			Position *int `json:"@position"`
		}

		if err := json.Unmarshal(raw, &fields); err != nil || fields.Position == nil {
			continue
		}

		out[key] = *fields.Position + wire.GridOffset
	}

	return out
}

// snapshotOn reads which grid positions one snapshot switches on.
//
// Only the one signal path this device has. A second processor's entries are
// named the same way as the first's, so reading both would put dsp1's block0
// on dsp0's grid.
func snapshotOn(
	raw json.RawMessage,
	at map[string]int,
) map[int]bool {
	if len(raw) == 0 {
		return nil
	}

	var byProcessor map[string]map[string]bool
	if err := json.Unmarshal(raw, &byProcessor); err != nil {
		return nil
	}

	entries, ok := byProcessor[processorKey]
	if !ok {
		return nil
	}

	out := map[int]bool{}

	for key, on := range entries {
		pos, ok := at[key]
		if !ok {
			continue
		}

		out[pos] = on
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// snapshotIndex reads the number a snapshot is stored under, or -1.
func snapshotIndex(
	key string,
) int {
	rest, ok := strings.CutPrefix(key, snapshotPrefix)
	if !ok {
		return -1
	}

	n, err := strconv.Atoi(rest)
	if err != nil {
		return -1
	}

	return n
}

// readInto reads one of a snapshot's fields, leaving it unset when absent.
//
// A preset omits a field rather than writing a null, and one that will not
// read is left unset for the same reason: a zero would claim the file said
// something it did not, and a zero is what would then reach the device.
func readInto[T any](
	raw json.RawMessage,
	target **T,
) {
	if len(raw) == 0 {
		return
	}

	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return
	}

	*target = &v
}
