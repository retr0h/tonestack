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
// Package catalogview reports what a device can do.
package catalogview

import (
	"sort"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// Filter narrows what List reports.
type Filter struct {
	// Category keeps only blocks of one kind — amp, cab, drive.
	Category string
	// Subcategory keeps only blocks Line 6 tags this way — Guitar, Bass.
	Subcategory string
	// Search keeps only blocks whose name or real-world gear mentions this.
	Search string
}

// List reads the blocks matching f.
func List(path string, f Filter) (result.Blocks, error) {
	c, err := catalog.Open(path)
	if err != nil {
		return result.Blocks{}, err
	}

	return result.Blocks{
		Device:  c.Device,
		Source:  c.Source,
		Total:   len(c.Blocks),
		Matched: match(c, f),
	}, nil
}

func match(c *catalog.Catalog, f Filter) []catalog.Block {
	out := make([]catalog.Block, 0, len(c.Blocks))

	for _, b := range c.Blocks {
		if f.Category != "" && !strings.EqualFold(string(b.Category), f.Category) {
			continue
		}

		if f.Subcategory != "" && !strings.EqualFold(b.Subcategory, f.Subcategory) {
			continue
		}

		if f.Search != "" && !mentions(b, f.Search) {
			continue
		}

		out = append(out, b)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// mentions reports whether a block's name or real-world gear contains term.
func mentions(b catalog.Block, term string) bool {
	// The same rule the resolver uses, so browsing for gear predicts whether
	// a rig naming it will build. Two matchers meant a search could find gear
	// that then failed to resolve, which is the worst way to learn the
	// difference.
	if b.Matches(term) {
		return true
	}

	// A model identifier is not gear, and nobody should write one in a rig.
	// It is still the fastest way to look one up when reading a preset.
	return strings.Contains(
		strings.ToLower(string(b.ID)), strings.ToLower(term))
}

// Show reads one block and everything it accepts.
func Show(path, id string) (catalog.Block, error) {
	c, err := catalog.Open(path)
	if err != nil {
		return catalog.Block{}, err
	}

	b, ok := c.Block(catalog.ModelID(id))
	if !ok {
		return catalog.Block{}, &NotFoundError{ID: id, Known: len(c.Blocks)}
	}

	return b, nil
}
