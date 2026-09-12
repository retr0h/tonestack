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

package sdk

import "github.com/retr0h/tonestack/pkg/sdk/result"

// What every operation answers with.
//
// Declared in result and named here, because the operations that build these
// live under internal and the Client that hands them back lives in this
// package: a type declared in either would make the other import it. Aliases
// rather than wrappers, so sdk.Listing and result.Listing are the same type
// and nothing has to be converted at the seam.
//
// A caller writes sdk.Listing and never sees any of that.
type (
	// Listing is what a setlist holds, slot by slot.
	Listing = result.Listing
	// Held is one slot and what is in it.
	Held = result.Held

	// Reading is one preset, read out of a slot or a file.
	Reading = result.Reading
	// Answer is a device's reply that did not decode as a preset.
	Answer = result.Answer
	// Written is a file this wrote, and what went into it.
	Written = result.Written

	// Change is what a write did.
	Change = result.Change
	// Action is what a write did to a slot.
	Action = result.Action
	// At is a slot and what it is called.
	At = result.At
	// Built is a preset compiled from a rig.
	Built = result.Built

	// Made is a preset built from a recipe.
	Made = result.Made
	// Added is a block put in the chain that the recipe did not name.
	Added = result.Added
	// Unfamiliar is a character term nothing defines.
	Unfamiliar = result.Unfamiliar
	// Moved is what a character term did to a parameter.
	Moved = result.Moved

	// Recipes is every rig under one directory.
	Recipes = result.Recipes
	// Recipe is one rig, and what reading it needs that the rig does not
	// carry.
	Recipe = result.Recipe
	// Variant is a rig that extends another.
	Variant = result.Variant
	// Scaffolded is a recipe this wrote.
	Scaffolded = result.Scaffolded

	// Attached is what is on the bus that this recognises.
	Attached = result.Attached
	// Attachment is one device on the bus.
	Attachment = result.Attachment

	// Blocks is what a device can do, narrowed to what was asked for.
	Blocks = result.Blocks

	// Measured is what the corpus recorded, and what was asked of it.
	Measured = result.Measured

	// Catalogued is what a catalog generation run produced.
	Catalogued = result.Catalogued
	// Counted is what a corpus measuring run produced.
	Counted = result.Counted
)

// What a write did, named so a caller can match on it.
const (
	// Copied is one slot's contents put into another.
	Copied = result.Copied
	// Swapped is two slots exchanged.
	Swapped = result.Swapped
	// Imported is a preset from a file put into a slot.
	Imported = result.Imported
	// Selected is a slot loaded the way a footswitch loads one, which
	// writes nothing.
	Selected = result.Selected
)

// DumpEnv names a file to write a device's raw answer to.
const DumpEnv = result.DumpEnv
