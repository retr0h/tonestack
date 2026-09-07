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

// Package chain is a signal chain compiled for one device, and the checks
// that say whether the device will load it.
//
// A Chain is the compiler's intermediate form, not a format. Nobody authors
// one, nothing exchanges one, and it has no schema: it exists between
// resolving a [RigSpec] against a catalog and writing a preset file. What is
// authored and shared is a RigSpec, which names real-world gear; what a
// device loads is a preset. This sits between them and is deliberately
// device-bound, holding one manufacturer's model identifiers and one
// manufacturer's parameter keys.
//
// [RigSpec]: https://github.com/retr0h/tonestack/blob/main/schemas/rigspec.openapi.yaml
package chain

import "github.com/retr0h/tonestack/pkg/catalog"

// Params are parameter values keyed by the device's own parameter key.
type Params map[string]catalog.ParamValue

// Block is one model in a chain, positioned.
type Block struct {
	// Model is a device-internal identifier such as HD2_AmpSVBeastNrm.
	// Validity is decided against a catalog, not here.
	Model catalog.ModelID
	// Params holds what every knob is set to.
	Params Params
	// DSP is which signal path this block occupies, from zero. Devices with
	// one path only ever use zero.
	DSP int
	// Pos is the position within its path. Positions on one path must form
	// the contiguous run 0..n-1.
	Pos int
	// Enabled says whether the block is doing anything. A bypassed block
	// still occupies its position and still costs DSP.
	Enabled bool
}

// Snapshot is one set of parameter overrides a device can switch between
// without changing preset.
//
// Device-bound, like everything else here: a RigSpec expresses the same idea
// as variants, in gear terms.
type Snapshot struct {
	// Name is what the device shows for it.
	Name string
	// Overrides are keyed by block index, as a string, because that is how
	// the preset format stores them.
	Overrides map[string]Params
}

// Chain is an ordered signal path for one device.
type Chain struct {
	// Name is what the preset will be called.
	Name string
	// Blocks are in the order the device runs them.
	Blocks []Block
	// Snapshots are the switchable parameter sets, if any.
	Snapshots []Snapshot
}

// BlockLookup is whatever can say what a model is.
//
// A catalog satisfies it. Taking the interface rather than the catalog keeps
// validation testable against a handful of blocks instead of six hundred.
type BlockLookup interface {
	Block(id catalog.ModelID) (catalog.Block, bool)
}

// Limits are what one device will accept.
type Limits struct {
	// MaxBlocks is the most blocks the device will hold.
	MaxBlocks int
	// Paths is how many signal paths the device has.
	//
	// An HX Stomp has one: no preset in a corpus of 714 uses a second, while
	// 73% of Helix Floor presets do. A chain that does not fit a device with
	// one path has nowhere to overflow to and must be refused.
	Paths int
	// ChipCeiling is how much of one processor a chain may occupy, in the
	// same units the catalog states a block's cost: percent. Line 6 records
	// an Ampeg SVT at 26.67, meaning a quarter of a processor.
	//
	// It sits below 100 deliberately. A chain that exactly fills a processor
	// in theory is a chain that fails to load in practice.
	ChipCeiling float64
}
