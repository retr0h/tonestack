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

import (
	"encoding/binary"
	"math"
)

// Encoding one replacement value.
//
// The device wrote the value being replaced, so its code is the best evidence
// available of what the device expects to read back. A replacement keeps that
// code wherever the new value fits it, and only widens when it does not. A
// parameter swapped for another parameter therefore leaves the section the
// length it was.

// encodeLike encodes value the way the byte it replaces was encoded.
func encodeLike(
	value any,
	like byte,
) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return []byte{codeNil}, nil
	case bool:
		if v {
			return []byte{codeTrue}, nil
		}

		return []byte{codeFalse}, nil
	case string:
		return encodeString(v, like), nil
	case float32:
		return encodeFloat(float64(v), like), nil
	case float64:
		return encodeFloat(v, like), nil
	}

	// asInt falls back to asUint, so this accepts every width of both.
	if n, ok := asInt(value); ok {
		return encodeInt(n, like), nil
	}

	return nil, &BadValueError{Value: value}
}

// encodeFloat writes a float.
//
// Every float in a captured preset is a float32. There are 140 of them and
// not one float64, so a float32 is what the device is given unless the value
// being replaced was wider.
func encodeFloat(
	v float64,
	like byte,
) []byte {
	if like == codeFloat64 {
		out := make([]byte, 9)
		out[0] = codeFloat64
		binary.BigEndian.PutUint64(out[1:], math.Float64bits(v))

		return out
	}

	out := make([]byte, 5)
	out[0] = codeFloat32
	binary.BigEndian.PutUint32(out[1:], math.Float32bits(float32(v)))

	return out
}

// encodeString writes a string, keeping the original's length prefix where it
// still holds.
func encodeString(
	v string,
	like byte,
) []byte {
	switch {
	case fits(like, codeStr8, len(v) <= math.MaxUint8),
		len(v) > int(fixstrMask) && len(v) <= math.MaxUint8:
		return prefixed(v, []byte{codeStr8, byte(len(v))})
	case fits(like, codeStr16, len(v) <= math.MaxUint16),
		len(v) > math.MaxUint8 && len(v) <= math.MaxUint16:
		head := make([]byte, 3)
		head[0] = codeStr16
		binary.BigEndian.PutUint16(head[1:], uint16(len(v)))

		return prefixed(v, head)
	case len(v) <= int(fixstrMask):
		return prefixed(v, []byte{minFixstr | byte(len(v))})
	default:
		head := make([]byte, 5)
		head[0] = codeStr32
		binary.BigEndian.PutUint32(head[1:], uint32(len(v)))

		return prefixed(v, head)
	}
}

// prefixed joins a length prefix to the string it counts.
func prefixed(
	v string,
	head []byte,
) []byte {
	out := make([]byte, 0, len(head)+len(v))
	out = append(out, head...)

	return append(out, v...)
}

// encodeInt writes an integer, keeping the original's width where it still
// holds and otherwise taking the narrowest form the value fits.
func encodeInt(
	v int64,
	like byte,
) []byte {
	if out, ok := sameWidth(v, like); ok {
		return out
	}

	switch {
	case v >= 0 && v <= maxPositiveFixint:
		return []byte{byte(v)}
	case v < 0 && v >= -32:
		return []byte{byte(v)}
	case v >= 0 && v <= math.MaxUint8:
		return []byte{codeUint8, byte(v)}
	case v >= math.MinInt8 && v <= math.MaxInt8:
		return []byte{codeInt8, byte(v)}
	case v >= 0 && v <= math.MaxUint16:
		return wide(codeUint16, uint64(v), 2)
	case v >= math.MinInt16 && v <= math.MaxInt16:
		return wide(codeInt16, uint64(v), 2)
	case v >= 0 && v <= math.MaxUint32:
		return wide(codeUint32, uint64(v), 4)
	case v >= math.MinInt32 && v <= math.MaxInt32:
		return wide(codeInt32, uint64(v), 4)
	case v >= 0:
		return wide(codeUint64, uint64(v), 8)
	default:
		return wide(codeInt64, uint64(v), 8)
	}
}

// sameWidth re-uses the original code when the new value still fits it.
func sameWidth(
	v int64,
	like byte,
) ([]byte, bool) {
	switch {
	case like <= maxPositiveFixint && v >= 0 && v <= maxPositiveFixint:
		return []byte{byte(v)}, true
	case like >= minNegativeFixint && v < 0 && v >= -32:
		return []byte{byte(v)}, true
	case like == codeUint8 && v >= 0 && v <= math.MaxUint8:
		return []byte{codeUint8, byte(v)}, true
	case like == codeInt8 && v >= math.MinInt8 && v <= math.MaxInt8:
		return []byte{codeInt8, byte(v)}, true
	case like == codeUint16 && v >= 0 && v <= math.MaxUint16:
		return wide(codeUint16, uint64(v), 2), true
	case like == codeInt16 && v >= math.MinInt16 && v <= math.MaxInt16:
		return wide(codeInt16, uint64(v), 2), true
	case like == codeUint32 && v >= 0 && v <= math.MaxUint32:
		return wide(codeUint32, uint64(v), 4), true
	case like == codeInt32 && v >= math.MinInt32 && v <= math.MaxInt32:
		return wide(codeInt32, uint64(v), 4), true
	case like == codeUint64 && v >= 0:
		return wide(codeUint64, uint64(v), 8), true
	case like == codeInt64:
		return wide(codeInt64, uint64(v), 8), true
	default:
		return nil, false
	}
}

// fits says whether the original code is the one asked about and the new
// value still suits it.
func fits(
	like, want byte,
	room bool,
) bool {
	return like == want && room
}

// wide writes a code and a big-endian body of the given size.
func wide(
	code byte,
	v uint64,
	size int,
) []byte {
	out := make([]byte, 1+size)
	out[0] = code

	switch size {
	case 2:
		binary.BigEndian.PutUint16(out[1:], uint16(v))
	case 4:
		binary.BigEndian.PutUint32(out[1:], uint32(v))
	default:
		binary.BigEndian.PutUint64(out[1:], v)
	}

	return out
}
