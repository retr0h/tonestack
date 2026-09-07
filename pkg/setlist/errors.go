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
package setlist

import (
	"errors"
	"fmt"
)

// Sentinels callers match with errors.Is.
var (
	// ErrNotASetlist is returned for a file that is not a setlist or bundle.
	ErrNotASetlist = errors.New("not a setlist")
	// ErrCorrupt is returned when the payload fails its own checksum.
	ErrCorrupt = errors.New("payload is corrupt")
	// ErrNoSuchSlot is returned for a slot outside the file.
	ErrNoSuchSlot = errors.New("no such slot")
)

// NotASetlistError names the schema that was found instead.
type NotASetlistError struct {
	Schema string
}

func (e *NotASetlistError) Error() string {
	if e.Schema == "" {
		return "not a setlist: no schema field"
	}

	return fmt.Sprintf("not a setlist: schema is %q", e.Schema)
}

func (*NotASetlistError) Unwrap() error { return ErrNotASetlist }

// CorruptError says how the payload failed its own declared checksum.
type CorruptError struct {
	Field string
	Want  uint64
	Got   uint64
}

func (e *CorruptError) Error() string {
	return fmt.Sprintf(
		"payload is corrupt: %s is %d, file declares %d",
		e.Field, e.Got, e.Want,
	)
}

func (*CorruptError) Unwrap() error { return ErrCorrupt }

// NoSuchSlotError names the address that was out of range.
type NoSuchSlotError struct {
	Setlist int
	Slot    int
	Have    int
}

func (e *NoSuchSlotError) Error() string {
	return fmt.Sprintf(
		"no such slot: setlist %d slot %d, but it holds %d slots",
		e.Setlist, e.Slot, e.Have,
	)
}

func (*NoSuchSlotError) Unwrap() error { return ErrNoSuchSlot }
