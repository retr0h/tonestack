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
	"errors"
	"fmt"
)

// Editing a section by decoding and re-encoding it does not work. Line 6
// choose an encoding per field rather than per value: across the captures a
// number that fits a fixint is written as uint8 282 times and as uint32 ten
// times, and every float is a float32 where the Go encoder writes float64.
// Re-encoding section 0 of a captured preset grows it from 588 bytes to 835.
//
// Nothing derives the original choice from the value, so the only edit that
// keeps the rest of a section intact is one that never touches it. Splice
// finds the bytes a value occupies and swaps those, and every byte outside
// that range survives because nothing reads it.

// ErrNoSuchPath is returned when a path names nothing in the section.
var ErrNoSuchPath = errors.New("no such path")

// NoSuchPathError says how far a path got before it ran out.
type NoSuchPathError struct {
	// Path is what was asked for.
	Path Path
	// Depth is how many steps resolved before the failure.
	Depth int
	// Why says what was there instead.
	Why string
}

func (e *NoSuchPathError) Error() string {
	return fmt.Sprintf("no such path %v: step %d %s", e.Path, e.Depth, e.Why)
}

func (*NoSuchPathError) Unwrap() error { return ErrNoSuchPath }

// ErrBadValue is returned when a replacement cannot be encoded.
var ErrBadValue = errors.New("cannot encode value")

// BadValueError names what could not be encoded.
type BadValueError struct {
	// Value is what the caller passed.
	Value any
}

func (e *BadValueError) Error() string {
	return fmt.Sprintf("cannot encode value of type %T", e.Value)
}

func (*BadValueError) Unwrap() error { return ErrBadValue }

// Path addresses one value inside a section.
//
// Every map key in a preset is an integer. There are 665 of them across the
// captures and not one is a string, so a step is an integer whichever kind of
// container it lands in, and the container decides whether it reads as a map
// key or an array index.
type Path []int

// Locate returns the half-open byte range the value at path occupies.
//
// An empty path is the section itself.
func Locate(
	body []byte,
	path Path,
) (int, int, error) {
	return locate(body, 0, path, path)
}

// SpliceRaw replaces one value with MessagePack bytes the caller supplies.
//
// This is the primitive. Everything outside the replaced range is copied
// through unread, so a section keeps the encoding Line 6 gave it.
func SpliceRaw(
	body []byte,
	path Path,
	raw []byte,
) ([]byte, error) {
	start, end, err := Locate(body, path)
	if err != nil {
		return nil, err
	}

	return replaceSpan(body, start, end, raw), nil
}

// replaceSpan swaps one byte range for another, copying the rest through.
func replaceSpan(
	body []byte,
	start, end int,
	raw []byte,
) []byte {
	out := make([]byte, 0, len(body)-(end-start)+len(raw))
	out = append(out, body[:start]...)
	out = append(out, raw...)

	return append(out, body[end:]...)
}

// Splice replaces one value, encoding it the way the device would.
//
// The replacement keeps the width the original was written with wherever the
// new value fits it, so swapping one parameter for another leaves the section
// the same length. A value needing more room widens to the narrowest form
// that holds it.
func Splice(
	body []byte,
	path Path,
	value any,
) ([]byte, error) {
	start, end, err := Locate(body, path)
	if err != nil {
		return nil, err
	}

	raw, err := encodeLike(value, body[start])
	if err != nil {
		return nil, err
	}

	return replaceSpan(body, start, end, raw), nil
}

// locate walks one step at a time, carrying the whole path for the error.
func locate(
	body []byte,
	at int,
	rest Path,
	full Path,
) (int, int, error) {
	depth := len(full) - len(rest)

	if at >= len(body) {
		return 0, 0, &NoSuchPathError{Path: full, Depth: depth, Why: "ran off the end"}
	}

	if len(rest) == 0 {
		end, err := skip(body, at)
		if err != nil {
			return 0, 0, err
		}

		return at, end, nil
	}

	if n, first, ok := mapAt(body, at); ok {
		return intoMap(body, first, n, rest, full)
	}

	if n, first, ok := arrayAt(body, at); ok {
		return intoArray(body, first, n, rest, full)
	}

	return 0, 0, &NoSuchPathError{
		Path:  full,
		Depth: depth,
		Why:   fmt.Sprintf("is not a container, it is code 0x%02x", body[at]),
	}
}

// intoMap finds the value the next step keys and carries on from there.
func intoMap(
	body []byte,
	at, n int,
	rest Path,
	full Path,
) (int, int, error) {
	for range n {
		key, next, err := readKey(body, at)
		if err != nil {
			return 0, 0, err
		}

		if key == rest[0] {
			return locate(body, next, rest[1:], full)
		}

		// Only a key that did not match costs a walk over its value.
		after, err := skip(body, next)
		if err != nil {
			return 0, 0, err
		}

		at = after
	}

	return 0, 0, &NoSuchPathError{
		Path:  full,
		Depth: len(full) - len(rest),
		Why:   fmt.Sprintf("key %d is not in the map", rest[0]),
	}
}

// intoArray finds the element the next step indexes.
func intoArray(
	body []byte,
	at, n int,
	rest Path,
	full Path,
) (int, int, error) {
	if rest[0] < 0 || rest[0] >= n {
		return 0, 0, &NoSuchPathError{
			Path:  full,
			Depth: len(full) - len(rest),
			Why:   fmt.Sprintf("index %d is outside an array of %d", rest[0], n),
		}
	}

	for range rest[0] {
		var err error
		if at, err = skip(body, at); err != nil {
			return 0, 0, err
		}
	}

	return locate(body, at, rest[1:], full)
}
