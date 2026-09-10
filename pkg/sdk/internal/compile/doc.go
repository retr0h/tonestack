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

// Package compile turns a rig into a preset, and a preset back into a rig.
//
// A rig names real-world gear; a device understands model identifiers. This is
// where the two meet. Resolving looks up what each named piece of gear
// corresponds to on this device, orders the result into a chain the hardware
// can represent, and sets every parameter to what Line 6 says it should be.
// Lowering writes that chain into a preset. Lifting reads a preset back out as
// a rig.
//
// Both directions, because a format that only reads one way is not an
// abstraction over anything. Nothing is lost either way. What a rig does not
// model as musical intent, meaning routing, snapshots, footswitch assignments
// and the metadata a preset carries, is recorded verbatim under `device`, so a
// rig lifted from a preset rebuilds that preset without the original file. A
// rig somebody typed carries none of it and is built into an untouched preset
// the device itself wrote.
//
// What it does not yet do is act on a rig's character lines. Those describe how
// a rig should sound, and turning them into parameter moves is the next piece
// of work. Until then a generated preset is the right gear at factory settings,
// which is a starting point rather than an answer.
package compile
