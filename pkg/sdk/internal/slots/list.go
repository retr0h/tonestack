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

package slots

import (
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/setlist"
)

// ListOptions says which setlist to list.
type ListOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// Path is the .hls or .hlb file to read.
	Path string
	// Setlist selects one setlist within a bundle.
	Setlist int
	// CatalogPath is the generated catalog, used to summarise each chain.
	CatalogPath string
	// All includes slots holding no blocks.
	All bool
}

// List prints every slot in a setlist.
//
// Empty slots are hidden by default. A device-written setlist always holds
// 128 of them and most are untouched, so listing them all buries the ones
// somebody actually made.
func List(opts ListOptions) (result.Listing, error) {
	doc, err := open(opts.Path)
	if err != nil {
		return result.Listing{}, err
	}

	if opts.Setlist < 0 || opts.Setlist >= len(doc.Setlists) {
		return result.Listing{}, &setlist.NoSuchSlotError{Setlist: opts.Setlist}
	}

	sl := doc.Setlists[opts.Setlist]
	held := make([]result.Held, 0, len(sl.Slots))

	for i := range sl.Slots {
		// A slot that fails to parse is still a slot. Reporting it as empty
		// beats refusing to list the hundred and twenty-seven around it.
		spec, _ := sl.Slots[i].Spec()

		held = append(held, result.Held{
			Slot:   i,
			Name:   sl.Slots[i].Meta.Name,
			Blocks: spec.Blocks,
		})
	}

	return result.Listing{Name: sl.Name(), Slots: held}, nil
}
