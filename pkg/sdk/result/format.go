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

import (
	"errors"
	"fmt"
)

// Format is what an export is written as: FormatRig or FormatPreset, and
// nothing else. An export refuses any other value, the zero value included,
// with ErrUnknownFormat.
//
// It satisfies pflag.Value, so a command line refuses a format that does not
// exist while it parses its flags rather than once the device is open.
type Format string

// The formats an export can take.
const (
	// FormatRig is a RigSpec: this project's own format, and the default.
	// Gear a person recognises, portable to other devices, and the thing
	// every other operation speaks.
	FormatRig Format = "rigspec"
	// FormatPreset is the device's own file. A faithful copy, carrying the
	// routing and snapshots a rig models but a person never chooses.
	FormatPreset Format = "hlx"
)

// ErrUnknownFormat reports a format name that is neither rigspec nor hlx.
var ErrUnknownFormat = errors.New("unknown format")

// String is the format's name.
func (f *Format) String() string { return string(*f) }

// Set takes a format by name, and refuses one that does not exist.
func (f *Format) Set(
	name string,
) error {
	switch Format(name) {
	case FormatRig, FormatPreset:
		*f = Format(name)

		return nil
	default:
		return fmt.Errorf("%w %q: use %s or %s", ErrUnknownFormat, name, FormatRig, FormatPreset)
	}
}

// Type is what a flag's usage calls the value. A format is typed as a name, so
// the usage reads the same as it did when the flag was a plain string.
func (*Format) Type() string { return "string" }
