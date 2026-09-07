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

// Line 6 encode the same number differently from one field to the next — a
// map key arrives as a wide uint16 where a fixint would do, and a decoder
// hands back whichever Go type fits. Reading a value therefore means
// accepting any of them rather than asserting one.

// asUint reads any unsigned-shaped value.
func asUint(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint64:
		return n, true
	case uint32:
		return uint64(n), true
	case uint16:
		return uint64(n), true
	case uint8:
		return uint64(n), true
	case int64:
		return uint64(n), n >= 0
	case int32:
		return uint64(n), n >= 0
	case int16:
		return uint64(n), n >= 0
	case int8:
		return uint64(n), n >= 0
	case int:
		return uint64(n), n >= 0
	default:
		return 0, false
	}
}

// asInt reads any signed-shaped value, which an error code is.
func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case int16:
		return int64(n), true
	case int8:
		return int64(n), true
	case int:
		return int64(n), true
	default:
		u, ok := asUint(v)

		return int64(u), ok
	}
}

// asString reads a string, trimming the terminator.
//
// Line 6's strings are C strings whose declared length counts the trailing
// NUL, so a name arrives one byte longer than it reads.
func asString(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}

	for len(s) > 0 && s[len(s)-1] == 0 {
		s = s[:len(s)-1]
	}

	return s, true
}

// asFloat reads a number a device sent in whatever width holds it.
//
// MessagePack carries a value in the narrowest form that fits, so the same
// field arrives as an integer in one preset and a float in another.
func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	}

	// asInt already falls back to the unsigned widths, so this covers every
	// integer a device can send.
	if n, ok := asInt(v); ok {
		return float64(n), true
	}

	return 0, false
}
