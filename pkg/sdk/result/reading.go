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

import (
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// Reading is one preset, read out of a slot or a file.
//
// Both halves are here because a caller asks for different things from the
// same read. Somebody looking at a preset wants the rig, which names gear the
// way a person would. Somebody copying one wants the document the device
// wrote, byte for byte. Producing both and letting the caller pick beats two
// operations that read the same slot twice.
type Reading struct {
	// Name is what the preset is called. A slot has one even when it holds
	// nothing, because a device names every slot whether or not anybody has
	// put anything in it.
	Name string
	// Doc is the preset itself. Nil when the slot holds nothing.
	Doc *preset.Document
	// Rig is what the preset describes. Zero when the slot holds nothing, or
	// when only the device's own document was asked for.
	Rig riggen.RigSpec
	// Answer is what a device replied with when the reply was not a preset.
	// Nil otherwise.
	Answer *Answer
}

// Empty says whether the slot holds a preset.
//
// A name is no guide: a device names every slot, so an untouched one still
// answers with whatever it shipped with. Only the document says.
func (r Reading) Empty() bool { return r.Doc == nil }

// DumpEnv names a file to write a device's raw answer to.
//
// Reading a preset off the hardware is the one call whose reply nobody has
// seen. Capturing it is what turns a guess about the wire format into a test,
// and it costs one plugged-in session rather than one per attempt.
const DumpEnv = "TONESTACK_USB_DUMP"

// Answer is a device's reply that did not decode as a preset.
//
// Kept and reported rather than discarded as a failure. A preset arrives as
// three concatenated MessagePack values and the models inside it are numbered
// by a scheme that does not index the catalog, so a reply nobody can read is
// how a protocol change becomes visible. Saying plainly what arrived beats
// printing a chain that would be wrong.
type Answer struct {
	// Model is what the device calls itself.
	Model string
	// Slot is the position that was read, from zero.
	Slot int
	// Shape is what arrived, described in whatever detail can be had.
	Shape string
}

// Written is a file this wrote, and what went into it.
type Written struct {
	// Slot is the position it came from, from zero.
	Slot int
	// Name is what the preset is called.
	Name string
	// Path is the file that was written.
	Path string
}
