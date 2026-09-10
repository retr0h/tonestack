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

import "github.com/retr0h/tonestack/pkg/sdk/catalog"

// Blocks is what a device can do, narrowed to what was asked for.
//
// The total is here alongside the matches because a search answering with
// four blocks means something different depending on whether the catalog
// holds six or six hundred.
type Blocks struct {
	// Device is what the catalog calls the hardware.
	Device string
	// Source says where the catalog came from. Empty when nothing recorded
	// it, which is different from a source nobody recognises.
	Source string
	// Total is how many blocks the catalog holds, before any filtering.
	Total int
	// Matched are the blocks that came through the filter.
	Matched []catalog.Block
}
