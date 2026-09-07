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

package catalog

import "strings"

// userIRPrefix marks the blocks that play a user's own impulse response.
const userIRPrefix = "HD2_ImpulseResponse"

// NeedsUserIR reports whether a model plays an impulse response the device
// owner loaded themselves.
//
// These blocks carry an Index rather than any audio: the impulse lives in a
// slot in the device's IR library, and what is in that slot differs from one
// instrument to the next. A preset naming slot 82 sounds like whatever its
// author had in slot 82 and like something else entirely on anybody else's
// device.
//
// Nothing generated should reach for one. Line 6's own cabinets, including
// the 92 impulse-response cabinets whose audio ships in the device, sound the
// same everywhere.
func NeedsUserIR(id ModelID) bool {
	return strings.HasPrefix(string(id), userIRPrefix)
}
