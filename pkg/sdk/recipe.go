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

import riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"

// Recipes is every rig under one directory.
type Recipes struct {
	// Dir is where they were read from.
	Dir string
	// Rigs are what was found, in the order they were read.
	Rigs []riggen.RigSpec
}

// Recipe is one rig, and what reading it needs that the rig does not carry.
type Recipe struct {
	// Rig is the rig itself.
	Rig riggen.RigSpec
	// Variants are the rigs that say they are a small change on this one.
	//
	// A rig cannot know this about itself. The link points the other way —
	// a variant names what it extends — so only the whole set can answer it,
	// and reading the characteristic rig is where somebody wants it.
	Variants []Variant
}

// Variant is a rig that extends another.
type Variant struct {
	// ID is what to ask for to read it.
	ID string
	// Name is what its subject is called.
	Name string
}

// Scaffolded is a recipe this wrote.
//
// What was written rather than what was asked for. The two are the same today,
// and saying so from the answer rather than from the request is what keeps a
// report honest if they ever stop being.
type Scaffolded struct {
	// ID is what to ask for to read it back.
	ID string
	// Name is what the subject is called.
	Name string
	// Instrument is what it is played on.
	Instrument string
	// Amp and Cab are the gear it names. Cab is empty when the amp carries
	// its own, which some models do.
	Amp string
	Cab string
	// Pedals are what else is in the chain, in order.
	Pedals []string
	// Path is the file that was written.
	Path string
}
