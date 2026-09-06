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

// Package rig describes a signal chain as intended, independent of the file
// format that will carry it. Every input path produces a Spec; everything that
// writes files consumes one.
package rig

import "github.com/retr0h/tonestack/pkg/catalog"

// Origin records how a rig was decided. It is distinct from
// catalog.Provenance, which records how a catalog entry was learned.
type Origin string

// How a rig was decided.
const (
	// OriginCurated matched a hand-authored recipe.
	OriginCurated Origin = "curated"
	// OriginLLM was generated against the catalog schema.
	OriginLLM Origin = "llm"
	// OriginAudio was derived from measurement.
	OriginAudio Origin = "audio"
)

// SpecBlock is one block placed in a chain: which model, at which position on
// which processor, with which parameters.
type SpecBlock struct {
	Model   catalog.ModelID               `json:"model"`
	Params  map[string]catalog.ParamValue `json:"params,omitempty"`
	DSP     int                           `json:"dsp"`
	Pos     int                           `json:"pos"`
	Enabled bool                          `json:"enabled"`
}

// Snapshot is a named set of parameter overrides, keyed by the index of the
// block in Spec.Blocks that each override applies to.
type Snapshot struct {
	Name      string                                `json:"name"`
	Overrides map[int]map[string]catalog.ParamValue `json:"overrides"`
}

// Spec is a complete signal chain, validated against a catalog before use.
type Spec struct {
	Name      string      `json:"name"`
	Blocks    []SpecBlock `json:"blocks"`
	Snapshots []Snapshot  `json:"snapshots,omitempty"`
	Origin    Origin      `json:"origin"`
}

// BlockLookup resolves a model identifier to its catalog entry.
// *catalog.Catalog satisfies it. Validators take this rather than a concrete
// catalog so a test can supply a handful of blocks without building a file.
type BlockLookup interface {
	Block(catalog.ModelID) (catalog.Block, bool)
}

// Limits are the ceilings one device imposes on a rig.
type Limits struct {
	// MaxBlocks is the most blocks the device will hold.
	MaxBlocks int
	// Chips is how many DSP processors the device has.
	Chips int
	// ChipCeiling is the fraction of one processor a rig may occupy. It is
	// deliberately below 1.0: a rig that exactly fills a chip in theory is a
	// rig that fails to load in practice.
	ChipCeiling float64
}
