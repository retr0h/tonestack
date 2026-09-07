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
	"errors"
	"fmt"
)

// Reading MessagePack without decoding it.
//
// A decoder turns bytes into Go values and loses which of the several legal
// encodings of a number the device chose. Splicing needs the opposite: where
// a value starts, where it ends, and nothing about what it means.

// ErrMalformed is returned when a section is not MessagePack.
var ErrMalformed = errors.New("malformed messagepack")

// MalformedError says where the trouble is.
type MalformedError struct {
	// At is the byte offset.
	At int
	// Why says what was wrong there.
	Why string
}

func (e *MalformedError) Error() string {
	return fmt.Sprintf("malformed messagepack at byte %d: %s", e.At, e.Why)
}

func (*MalformedError) Unwrap() error { return ErrMalformed }

// The format's fixed codes. The ranges are handled by comparison.
const (
	codeNil     = 0xc0
	codeFalse   = 0xc2
	codeTrue    = 0xc3
	codeBin8    = 0xc4
	codeBin16   = 0xc5
	codeBin32   = 0xc6
	codeExt8    = 0xc7
	codeExt16   = 0xc8
	codeExt32   = 0xc9
	codeFloat32 = 0xca
	codeFloat64 = 0xcb
	codeUint8   = 0xcc
	codeUint16  = 0xcd
	codeUint32  = 0xce
	codeUint64  = 0xcf
	codeInt8    = 0xd0
	codeInt16   = 0xd1
	codeInt32   = 0xd2
	codeInt64   = 0xd3
	codeStr8    = 0xd9
	codeStr16   = 0xda
	codeStr32   = 0xdb
	codeArray16 = 0xdc
	codeArray32 = 0xdd
	codeMap16   = 0xde
	codeMap32   = 0xdf
)

// The ranges a single byte encodes on its own.
const (
	maxPositiveFixint = 0x7f
	minNegativeFixint = 0xe0
	minFixmap         = 0x80
	maxFixmap         = 0x8f
	minFixarray       = 0x90
	maxFixarray       = 0x9f
	minFixstr         = 0xa0
	maxFixstr         = 0xbf
	fixMask           = 0x0f
	fixstrMask        = 0x1f
)

// skip returns the offset one past the value starting at at.
func skip(
	body []byte,
	at int,
) (int, error) {
	if at >= len(body) {
		return 0, &MalformedError{At: at, Why: "no value here"}
	}

	if n, first, ok := mapAt(body, at); ok {
		return skipRun(body, first, n*2)
	}

	if n, first, ok := arrayAt(body, at); ok {
		return skipRun(body, first, n)
	}

	return skipScalar(body, at)
}

// skipRun steps over n consecutive values.
func skipRun(
	body []byte,
	at, n int,
) (int, error) {
	for range n {
		var err error
		if at, err = skip(body, at); err != nil {
			return 0, err
		}
	}

	return at, nil
}

// skipScalar steps over anything that holds no other value.
func skipScalar(
	body []byte,
	at int,
) (int, error) {
	if at >= len(body) {
		return 0, &MalformedError{At: at, Why: "no value here"}
	}

	c := body[at]

	if c <= maxPositiveFixint || c >= minNegativeFixint {
		return at + 1, nil
	}

	if c >= minFixstr && c <= maxFixstr {
		return bounded(body, at, at+1+int(c&fixstrMask))
	}

	if width, ok := fixedWidth(c); ok {
		return bounded(body, at, at+width)
	}

	if header, ok := variableWidth(c); ok {
		n, err := length(body, at+1, header)
		if err != nil {
			return 0, err
		}

		return bounded(body, at, at+1+header+n+extra(c))
	}

	return 0, &MalformedError{At: at, Why: fmt.Sprintf("unknown code 0x%02x", c)}
}

// fixedWidth gives the whole size of a value whose length is in its code.
func fixedWidth(c byte) (int, bool) {
	switch c {
	case codeNil, codeFalse, codeTrue:
		return 1, true
	case codeUint8, codeInt8:
		return 2, true
	case codeUint16, codeInt16:
		return 3, true
	case codeUint32, codeInt32, codeFloat32:
		return 5, true
	case codeUint64, codeInt64, codeFloat64:
		return 9, true
	case 0xd4:
		return 3, true
	case 0xd5:
		return 4, true
	case 0xd6:
		return 6, true
	case 0xd7:
		return 10, true
	case 0xd8:
		return 18, true
	default:
		return 0, false
	}
}

// variableWidth gives how many bytes hold the length of a counted value.
func variableWidth(c byte) (int, bool) {
	switch c {
	case codeBin8, codeStr8, codeExt8:
		return 1, true
	case codeBin16, codeStr16, codeExt16:
		return 2, true
	case codeBin32, codeStr32, codeExt32:
		return 4, true
	default:
		return 0, false
	}
}

// extra is the type byte the ext family carries after its length.
func extra(c byte) int {
	if c == codeExt8 || c == codeExt16 || c == codeExt32 {
		return 1
	}

	return 0
}

// length reads a big-endian count of the given width.
func length(
	body []byte,
	at, width int,
) (int, error) {
	if at+width > len(body) {
		return 0, &MalformedError{At: at, Why: "length runs past the end"}
	}

	switch width {
	case 1:
		return int(body[at]), nil
	case 2:
		return int(binary.BigEndian.Uint16(body[at:])), nil
	default:
		return int(binary.BigEndian.Uint32(body[at:])), nil
	}
}

// bounded rejects a value claiming more bytes than the section has.
func bounded(
	body []byte,
	at, end int,
) (int, error) {
	if end > len(body) || end < at {
		return 0, &MalformedError{At: at, Why: "value runs past the end"}
	}

	return end, nil
}

// mapAt reports a map's pair count and where its first key starts.
func mapAt(
	body []byte,
	at int,
) (int, int, bool) {
	c := body[at]

	if c >= minFixmap && c <= maxFixmap {
		return int(c & fixMask), at + 1, true
	}

	return counted(body, at, codeMap16, codeMap32)
}

// arrayAt reports an array's length and where its first element starts.
func arrayAt(
	body []byte,
	at int,
) (int, int, bool) {
	c := body[at]

	if c >= minFixarray && c <= maxFixarray {
		return int(c & fixMask), at + 1, true
	}

	return counted(body, at, codeArray16, codeArray32)
}

// counted reads the two wide forms a map or an array shares the shape of.
func counted(
	body []byte,
	at int,
	wide16, wide32 byte,
) (int, int, bool) {
	switch body[at] {
	case wide16:
		n, err := length(body, at+1, 2)

		return n, at + 3, err == nil
	case wide32:
		n, err := length(body, at+1, 4)

		return n, at + 5, err == nil
	default:
		return 0, 0, false
	}
}

// readKey reads a map key and says where its value starts.
//
// Every key in a preset is an integer, so this reads no other kind.
func readKey(
	body []byte,
	at int,
) (int, int, error) {
	end, err := skipScalar(body, at)
	if err != nil {
		return 0, 0, err
	}

	c := body[at]

	switch {
	case c <= maxPositiveFixint:
		return int(c), end, nil
	case c >= minNegativeFixint:
		return int(int8(c)), end, nil
	case c == codeUint8:
		return int(body[at+1]), end, nil
	case c == codeInt8:
		return int(int8(body[at+1])), end, nil
	case c == codeUint16:
		return int(binary.BigEndian.Uint16(body[at+1:])), end, nil
	case c == codeInt16:
		return int(int16(binary.BigEndian.Uint16(body[at+1:]))), end, nil
	case c == codeUint32:
		return int(binary.BigEndian.Uint32(body[at+1:])), end, nil
	case c == codeInt32:
		return int(int32(binary.BigEndian.Uint32(body[at+1:]))), end, nil
	default:
		return 0, 0, &MalformedError{
			At:  at,
			Why: fmt.Sprintf("key code 0x%02x is not an integer", c),
		}
	}
}
