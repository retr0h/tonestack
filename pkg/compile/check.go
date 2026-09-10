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
	"maps"
	"slices"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// check reports what a rig claims that this device cannot supply.
//
// The contract cannot do this. A colour, a parameter name and a device name
// are all valid strings, and whether they are valid values is a question about
// the catalog in hand — which is why these run here, beside the gear names,
// rather than in the package that defines the device-independent format.
//
// The blocks are the chain as resolved, because a controller names the block
// it moves by position and the parameter by name, and only the model sitting
// at that position says whether the name is one of its own.
func check(spec riggen.RigSpec, blocks []chain.Block, cat *catalog.Catalog) error {
	// Every complaint at once. A rig with four bad colours in it took four
	// runs to fix when this reported the first one, and each run hid the
	// next. errors.Is and errors.As reach through a join, so a caller
	// matching on ErrNoSuchValue still matches.
	return errors.Join(
		checkTarget(spec, cat),
		checkFootswitches(spec, cat),
		checkControllers(spec, blocks, cat),
	)
}

// checkTarget refuses a rig built for another device.
func checkTarget(spec riggen.RigSpec, cat *catalog.Catalog) error {
	if spec.Target == nil || spec.Target.Device == nil || *spec.Target.Device == "" {
		return nil
	}

	if strings.EqualFold(*spec.Target.Device, cat.Device) {
		return nil
	}

	return &NoSuchValueError{
		Field: "target.device",
		Value: *spec.Target.Device,
		Near:  []string{cat.Device},
		Whole: true,
	}
}

// checkFootswitches refuses a colour the device cannot light.
//
// The device has twelve, and the catalog carries their names. A rig naming a
// thirteenth describes a switch nobody will see.
func checkFootswitches(spec riggen.RigSpec, cat *catalog.Catalog) error {
	if spec.Footswitches == nil || len(cat.LEDColours) == 0 {
		return nil
	}

	out := []error(nil)

	for i, fs := range *spec.Footswitches {
		if fs.Led == nil || *fs.Led == "" {
			continue
		}

		if slicesContainFold(cat.LEDColours, *fs.Led) {
			continue
		}

		hits, whole := near(cat.LEDColours, *fs.Led)

		out = append(out, &NoSuchValueError{
			Field: fmt.Sprintf("footswitches[%d].led", i),
			Value: *fs.Led,
			Near:  hits,
			Whole: whole,
		})
	}

	return errors.Join(out...)
}

// checkControllers refuses a parameter the block it moves does not have.
//
// A device stores the parameter's place in the model's own list rather than
// its name, so a name nothing matches is written as a position — and the
// controller ends up moving whatever happens to sit there.
func checkControllers(
	spec riggen.RigSpec,
	blocks []chain.Block,
	cat *catalog.Catalog,
) error {
	if spec.Controllers == nil {
		return nil
	}

	out := []error(nil)

	for i, c := range *spec.Controllers {
		// By position along the path rather than by place in the list. A rig
		// read off a device numbers its blocks the way the device lays them
		// out, and a chain that states its positions leaves gaps in them.
		at, ok := blockAt(blocks, c.Block)
		if !ok {
			out = append(out, &NoSuchBlockError{
				Field: fmt.Sprintf("controllers[%d].block", i),
				Block: c.Block,
				Have:  len(blocks),
			})

			continue
		}

		b, ok := cat.Block(at.Model)
		if !ok {
			continue
		}

		if _, ok := b.Params[c.Parameter]; ok {
			continue
		}

		// Sorted, because they come out of a map and an error message that
		// reads differently every run is one nobody can test or quote.
		names := slices.Sorted(maps.Keys(b.Params))

		hits, whole := near(names, c.Parameter)

		out = append(out, &NoSuchValueError{
			Field: fmt.Sprintf("controllers[%d].parameter", i),
			Value: c.Parameter,
			Near:  hits,
			Whole: whole,
		})
	}

	return errors.Join(out...)
}

// blockAt finds the block sitting at a position along the path.
func blockAt(blocks []chain.Block, position int) (chain.Block, bool) {
	for _, b := range blocks {
		if b.Pos == position {
			return b, true
		}
	}

	return chain.Block{}, false
}

// slicesContainFold reports whether a value is in a list, ignoring case.
func slicesContainFold(all []string, want string) bool {
	for _, s := range all {
		if strings.EqualFold(s, want) {
			return true
		}
	}

	return false
}

// near suggests the values closest to what was asked for, and says whether
// what it returned is everything there is.
//
// A short list is worth printing whole when nothing matched: twelve colours
// answer the question outright, where forty parameter names only bury it.
//
// For names: a colour, a parameter, a piece of gear. Somebody types part of
// one or mistypes its first letters, so a prefix and a substring are what
// find it. A phrase in the wrong order is a different problem and closest,
// beside this, is what solves it.
func near(all []string, want string) (hits []string, whole bool) {
	want = strings.ToLower(want)

	for _, s := range all {
		low := strings.ToLower(s)
		if strings.HasPrefix(low, want[:min(len(want), 3)]) || strings.Contains(low, want) {
			hits = append(hits, s)
		}
	}

	if len(hits) == 0 {
		if len(all) <= nearWhole {
			return all, true
		}

		return nil, false
	}

	if len(hits) > nearMost {
		hits = hits[:nearMost]
	}

	return hits, false
}

// How much of a list is worth printing when nothing matched, and how many
// near misses are worth naming when something did.
const (
	nearWhole = 12
	nearMost  = 5
)
