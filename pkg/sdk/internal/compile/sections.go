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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// ErrSectionsAndSnapshots reports a rig carrying both.
//
// Snapshots are what a device stored and sections are what somebody wants.
// Building from both would mean quietly picking one.
var ErrSectionsAndSnapshots = errors.New(
	"a rig carries sections or snapshots, not both")

// ErrTooManySections reports more sections than the device has snapshots.
var ErrTooManySections = errors.New("more sections than this device has snapshots")

// ErrSectionContradicts reports a role a section both plays and bypasses.
var ErrSectionContradicts = errors.New("a section plays and bypasses the same role")

// Sections writes a rig's song sections into the preset's snapshots.
//
// Each section becomes the snapshot at its place in the list: its name, and
// whether each block plays. A role names every block in the chain the
// catalog files under it, and a block no section mentions keeps the state the
// chain gives it.
//
// Everything is checked before anything is written, so a rig that will not
// build leaves the preset as it was.
func Sections(
	doc *preset.Document,
	spec rig.Spec,
	blocks []chain.Block,
	cat *catalog.Catalog,
) error {
	if spec.Sections == nil {
		return nil
	}

	if spec.Snapshots != nil {
		return ErrSectionsAndSnapshots
	}

	sections := *spec.Sections

	// Counted from the preset rather than assumed, because the number is the
	// device's: the preset underneath was written by one.
	room := 0

	for key := range doc.Data.Tone {
		if snapshotIndex(key) >= 0 {
			room++
		}
	}

	if len(sections) > room {
		return fmt.Errorf("%w: %d sections, and this device has %d",
			ErrTooManySections, len(sections), room)
	}

	roles := rolesOf(blocks, cat)
	states := make([][]bool, len(sections))
	errs := []error(nil)

	for i, sec := range sections {
		state := make([]bool, len(blocks))
		for b, block := range blocks {
			state[b] = block.Enabled
		}

		errs = append(errs,
			engage(state, roles, sec.Play, true, fmt.Sprintf("sections[%d].play", i)),
			engage(state, roles, sec.Bypass, false, fmt.Sprintf("sections[%d].bypass", i)),
			contradiction(sec, i))

		states[i] = state
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	for i, sec := range sections {
		key := snapshotPrefix + strconv.Itoa(i)
		doc.Data.Tone[key] = sectionEntry(doc.Data.Tone[key], sec.Name, blocks, states[i])
	}

	return nil
}

// rolesOf says what each block in a chain is for, as the catalog files it.
//
// A model the catalog does not carry has no role, so no section can name it.
func rolesOf(
	blocks []chain.Block,
	cat *catalog.Catalog,
) []rig.Role {
	out := make([]rig.Role, len(blocks))

	for i, b := range blocks {
		if found, ok := cat.Block(b.Model); ok {
			out[i] = rig.Role(found.Category)
		}
	}

	return out
}

// engage sets every block with one of the named roles to on, and refuses a
// role the chain has no block for.
//
// A section that turns on a drive the rig does not have describes a sound
// the preset cannot make, and saying nothing would build it anyway.
func engage(
	state []bool,
	roles []rig.Role,
	named *[]rig.Role,
	on bool,
	field string,
) error {
	if named == nil {
		return nil
	}

	out := []error(nil)

	for _, want := range *named {
		found := false

		for b, role := range roles {
			if role == want {
				state[b] = on
				found = true
			}
		}

		if !found {
			out = append(out, &NoSuchValueError{
				Field: field,
				Value: string(want),
				Near:  present(roles),
				Whole: true,
			})
		}
	}

	return errors.Join(out...)
}

// present lists the roles a chain has, once each and in order.
func present(
	roles []rig.Role,
) []string {
	out := []string(nil)

	for _, role := range roles {
		if role != "" && !slices.Contains(out, string(role)) {
			out = append(out, string(role))
		}
	}

	slices.Sort(out)

	return out
}

// contradiction refuses a role a section both plays and bypasses.
func contradiction(
	sec rig.Section,
	i int,
) error {
	if sec.Play == nil || sec.Bypass == nil {
		return nil
	}

	out := []error(nil)

	for _, role := range *sec.Play {
		if slices.Contains(*sec.Bypass, role) {
			out = append(out, fmt.Errorf("sections[%d]: %w: %s", i, ErrSectionContradicts, role))
		}
	}

	return errors.Join(out...)
}

// sectionEntry builds one snapshot from a section.
//
// Written over the snapshot already there rather than in place of it, so the
// tempo, colour and anything else the device stored in that slot survive.
func sectionEntry(
	existing preset.Tone,
	name string,
	blocks []chain.Block,
	state []bool,
) preset.Tone {
	entry := preset.Tone{}
	for key, raw := range existing {
		entry[key] = raw
	}

	named := true
	put(entry, snapName, &name)
	put(entry, snapNamed, &named)
	put(entry, snapValid, &named)

	// Keyed the way the preset keys the blocks themselves: by path, then by
	// the position a block is stored under. What else a path records, such as
	// whether its split is on, stays; the blocks are the section's to decide,
	// including dropping one the chain no longer holds.
	paths := map[string]map[string]bool{}
	var kept *map[string]map[string]bool
	decode(entry[snapBlocks], &kept)

	if kept != nil {
		for path, states := range *kept {
			paths[path] = map[string]bool{}

			for key, on := range states {
				if !isBlockKey(key) {
					paths[path][key] = on
				}
			}
		}
	}

	for b, block := range blocks {
		path := processorKey(block.DSP)
		if paths[path] == nil {
			paths[path] = map[string]bool{}
		}

		paths[path]["block"+strconv.Itoa(block.Pos)] = state[b]
	}

	put(entry, snapBlocks, &paths)

	return entry
}

// processorKey names the tone entry holding one signal path.
func processorKey(
	dsp int,
) string {
	return "dsp" + strconv.Itoa(dsp)
}

// isBlockKey says whether a key in a snapshot's path names a block.
func isBlockKey(
	key string,
) bool {
	rest, ok := strings.CutPrefix(key, "block")
	if !ok {
		return false
	}

	_, err := strconv.Atoi(rest)

	return err == nil
}
