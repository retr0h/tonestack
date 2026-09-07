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
	"sort"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// snapshotPrefix marks a tone entry holding a snapshot.
const snapshotPrefix = "snapshot"

// Keys a preset stores a snapshot under. A device owns these names.
const (
	snapName        = "@name"
	snapTempo       = "@tempo"
	snapLED         = "@ledcolor"
	snapPedal       = "@pedalstate"
	snapNamed       = "@custom_name"
	snapValid       = "@valid"
	snapBlocks      = "blocks"
	snapControllers = "controllers"
)

// snapshotsOf reads a preset's snapshots, in the order the device numbers
// them.
//
// Modelled rather than carried as device state, because a snapshot is a
// musical decision: which blocks are on, at what tempo, under what name. What
// somebody put on a footswitch is part of the rig.
func snapshotsOf(doc *preset.Document) *[]riggen.Snapshot {
	keys := make([]string, 0, len(doc.Data.Tone))

	for key := range doc.Data.Tone {
		if strings.HasPrefix(key, snapshotPrefix) && snapshotIndex(key) >= 0 {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return nil
	}

	sort.Slice(keys, func(i, j int) bool {
		return snapshotIndex(keys[i]) < snapshotIndex(keys[j])
	})

	out := make([]riggen.Snapshot, 0, len(keys))

	for _, key := range keys {
		out = append(out, snapshotOf(doc.Data.Tone[key]))
	}

	return &out
}

// snapshotOf reads one snapshot entry.
func snapshotOf(entry preset.Tone) riggen.Snapshot {
	var out riggen.Snapshot

	decode(entry[snapName], &out.Name)
	decode(entry[snapTempo], &out.Tempo)
	decode(entry[snapLED], &out.Led)
	decode(entry[snapPedal], &out.Pedal)
	decode(entry[snapNamed], &out.Named)
	decode(entry[snapValid], &out.Valid)

	if raw, ok := entry[snapBlocks]; ok {
		out.Blocks = &raw
	}

	if raw, ok := entry[snapControllers]; ok {
		out.Controllers = &raw
	}

	// Whatever this does not model. A handful of presets record commands
	// here, and a format that dropped them would not be lossless the first
	// time Line 6 added a field.
	rest := map[string]json.RawMessage{}

	for key, raw := range entry {
		if !modelled[key] {
			rest[key] = raw
		}
	}

	if len(rest) > 0 {
		out.Rest = &rest
	}

	return out
}

// modelled names the snapshot fields this package reads by name.
var modelled = map[string]bool{
	snapName: true, snapTempo: true, snapLED: true, snapPedal: true,
	snapNamed: true, snapValid: true, snapBlocks: true, snapControllers: true,
}

// restoreSnapshots writes a rig's snapshots back as the device stores them.
//
// Numbered from zero in the order the rig lists them, which is the order they
// were read in. A device names them snapshot0 upward and nothing else refers
// to them by name.
func restoreSnapshots(doc *preset.Document, snapshots []riggen.Snapshot) {
	for i, snap := range snapshots {
		entry := preset.Tone{}

		put(entry, snapName, snap.Name)
		put(entry, snapTempo, snap.Tempo)
		put(entry, snapLED, snap.Led)
		put(entry, snapPedal, snap.Pedal)
		put(entry, snapNamed, snap.Named)
		put(entry, snapValid, snap.Valid)

		if snap.Blocks != nil {
			entry[snapBlocks] = *snap.Blocks
		}

		if snap.Controllers != nil {
			entry[snapControllers] = *snap.Controllers
		}

		if snap.Rest != nil {
			for key, raw := range *snap.Rest {
				entry[key] = raw
			}
		}

		doc.Data.Tone[snapshotPrefix+strconv.Itoa(i)] = entry
	}
}

// snapshotIndex reads the number a snapshot is stored under, or -1.
func snapshotIndex(key string) int {
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

// decode reads one of a snapshot's fields, leaving it unset when absent.
//
// A preset omits fields rather than writing nulls, and a field this cannot
// read is left unset for the same reason: writing a zero would claim the
// device said something it did not.
func decode[T any](raw json.RawMessage, target **T) {
	if len(raw) == 0 {
		return
	}

	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return
	}

	*target = &v
}

// put writes a field back, or writes nothing when the rig has none.
func put[T any](entry preset.Tone, key string, v *T) {
	if v == nil {
		return
	}

	// A value that came out of JSON goes back into it, so this cannot fail.
	body, _ := json.Marshal(*v)
	entry[key] = body
}
