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
package compile

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNoSuchGear reports gear no model on this device emulates.
var ErrNoSuchGear = errors.New("no model emulates that gear")

// NoSuchGearError names the gear and where it was looked for.
type NoSuchGearError struct {
	// Gear is what the recipe named, e.g. "Ampeg SVT".
	Gear string
	// Kind is the sort of block wanted — amp, cab, drive.
	Kind string
	// Instrument is the half of the catalog searched, when one was.
	Instrument string
}

// Error implements the error interface.
func (e *NoSuchGearError) Error() string {
	where := "this device's catalog"
	if e.Instrument != "" {
		where = fmt.Sprintf("this device's %s %ss", e.Instrument, e.Kind)
	}

	return fmt.Sprintf("no %s in %s emulates %q", e.Kind, where, e.Gear)
}

// Unwrap returns ErrNoSuchGear so callers can match with errors.Is.
func (*NoSuchGearError) Unwrap() error { return ErrNoSuchGear }

// ErrNoSuchValue reports a rig naming something this device does not have.
//
// Distinct from gear, which every rig names and most catalogs can supply. This
// is a colour, a parameter or a device name: valid strings, and valid values
// only against the catalog in hand.
var ErrNoSuchValue = errors.New("this device has no such value")

// NoSuchValueError names the field, what it said, and what was available.
type NoSuchValueError struct {
	// Field is where it was said, e.g. "footswitches[0].led".
	Field string
	// Value is what the rig named.
	Value string
	// Near is what the device has, or the closest of it.
	Near []string
	// Whole says Near is everything there is rather than a shortlist, which
	// is a different sentence: a device has twelve colours and the whole list
	// is the answer, where nine parameter names are a guess at the one meant.
	Whole bool
}

// Error implements the error interface.
func (e *NoSuchValueError) Error() string {
	msg := fmt.Sprintf("%s: this device has no %q", e.Field, e.Value)

	switch {
	case len(e.Near) == 0:
	case e.Whole:
		msg += "\n  it has: " + strings.Join(e.Near, ", ")
	default:
		msg += "\n  did you mean: " + strings.Join(e.Near, ", ")
	}

	return msg
}

// Unwrap returns ErrNoSuchValue so callers can match with errors.Is.
func (*NoSuchValueError) Unwrap() error { return ErrNoSuchValue }

// ErrNoSuchBlock reports a rig pointing at a block its own chain does not
// have.
var ErrNoSuchBlock = errors.New("the chain has no such block")

// NoSuchBlockError names the field, the position it asked for, and how many
// blocks there are.
type NoSuchBlockError struct {
	// Field is where it was said, e.g. "controllers[0].block".
	Field string
	// Block is the position along the path the rig named.
	Block int
	// Have is how many blocks the chain holds.
	Have int
}

// Error implements the error interface.
func (e *NoSuchBlockError) Error() string {
	blocks := "blocks"
	if e.Have == 1 {
		blocks = "block"
	}

	return fmt.Sprintf("%s: no block sits at position %d, and the chain holds %d %s",
		e.Field, e.Block, e.Have, blocks)
}

// Unwrap returns ErrNoSuchBlock so callers can match with errors.Is.
func (*NoSuchBlockError) Unwrap() error { return ErrNoSuchBlock }
