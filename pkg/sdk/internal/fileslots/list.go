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

package fileslots

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/internal/setlist"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// List answers with every slot in one setlist of a file.
//
// Every slot, including the ones holding nothing. Which to show is the
// renderer's decision: a device-written setlist always holds 128 slots and most
// are untouched, and whether to bury the ones somebody made is a matter of how
// the answer is drawn.
//
// A catalog is not read. A listing says which blocks a slot holds, and naming
// them is the renderer's job.
func (*Flows) List(
	ctx context.Context,
	path string,
	at int,
) (result.Listing, error) {
	doc, err := open(ctx, path)
	if err != nil {
		return result.Listing{}, err
	}

	if at < 0 || at >= len(doc.Setlists) {
		return result.Listing{}, &setlist.NoSuchSlotError{Setlist: at}
	}

	sl := doc.Setlists[at]
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
