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

package wire

import "github.com/vmihailenco/msgpack/v5"

// Exposed to the package's own tests. A MessagePack decoder hands back
// whichever Go type fits the value it read, and these accept any of them —
// which is only checkable by supplying each one.
var (
	AsUint   = asUint
	AsInt    = asInt
	AsString = asString
)

// NewDocument returns a copy of one holding only the named sections, so a
// preset with fewer than a device writes can be tested.
func NewDocument(from *Document, keep []int8) *Document {
	out := &Document{
		magic:    from.magic,
		table:    from.table,
		sections: map[int8]msgpack.RawMessage{},
	}

	for _, key := range keep {
		if body, ok := from.sections[key]; ok {
			out.order = append(out.order, key)
			out.sections[key] = body
		}
	}

	return out
}
