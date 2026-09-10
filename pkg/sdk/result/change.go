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

package result

// Change is what a write did.
//
// A device has no undo, so what a write says it did is the only record of it.
// Every field here is something somebody needs after the fact: which slot lost
// what it held, what was kept before that happened, and where the result went.
type Change struct {
	// Action says what was done.
	Action Action
	// From is the slot it came from. Nil when the source was a file rather
	// than another slot.
	From *At
	// To is the slot it went to.
	To At
	// Replaced is what the destination held before, which nothing else
	// records once the write has happened.
	Replaced string
	// Kept are the backup files written before anything was overwritten,
	// in the order they were made. Empty when there was nothing to keep,
	// which is what an untouched slot answers with.
	Kept []string
	// Mismatch says the preset was made for a different device, so it may
	// not load. Written anyway: the device identifier is not a reliable
	// refusal, and somebody who moved a preset on purpose is owed the
	// attempt rather than a veto.
	Mismatch bool
	// Path is the file the result went to. Empty when the write went to a
	// device, which is written in place and has no file.
	Path string
}

// OnDevice says whether the write went to hardware rather than a file.
//
// The two differ in what can be said afterwards. A file has somewhere to go
// back to; a device has only whatever was kept on the way past.
func (c Change) OnDevice() bool { return c.Path == "" }

// Action is what a write did to a slot.
type Action string

const (
	// Copied is one slot's contents put into another.
	Copied Action = "copied"
	// Swapped is two slots exchanged.
	Swapped Action = "swapped"
	// Imported is a preset from a file put into a slot.
	Imported Action = "written"
	// Selected is a slot loaded the way a footswitch loads one, which
	// writes nothing.
	Selected Action = "selected"
)

// At is a slot and what it is called.
type At struct {
	// Slot is the position, from zero. The label a pedal prints is
	// slot.Label of it, which is a rendering decision rather than data.
	Slot int
	// Name is what the slot is called.
	Name string
}

// Built is a preset compiled from a rig.
//
// The block count is here because it is the one number that says whether the
// rig somebody wrote turned into the chain they meant. A preset with two
// blocks where they described five is written, loads, and is wrong.
type Built struct {
	// Name is what the preset is called, which is the rig's subject.
	Name string
	// Blocks is how many are in the chain.
	Blocks int
	// Path is the file that was written.
	Path string
}
