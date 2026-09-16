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

import (
	"errors"
	"fmt"
	"strings"
)

// ErrBadParam reports a parameter that does not exist on a block, or whose
// value does not fit the declared type or range.
var ErrBadParam = errors.New("invalid parameter")

// ErrNoDevice reports a device this binary ships no catalog for.
var ErrNoDevice = errors.New("no built-in catalog")

// UnknownDeviceError names a device nothing here ships a catalog for.
//
// With the ones it does, because the answer to "that is not a device I carry"
// is the list of devices it carries.
type UnknownDeviceError struct {
	// Name is what was asked for.
	Name string
	// Known is every device a catalog ships for.
	Known []string
}

// Error implements the error interface.
func (e *UnknownDeviceError) Error() string {
	return fmt.Sprintf("no catalog is built in for %q: this binary carries %s",
		e.Name, strings.Join(e.Known, ", "))
}

// Unwrap returns ErrNoDevice so callers can match with errors.Is.
func (*UnknownDeviceError) Unwrap() error { return ErrNoDevice }

// NoDeviceError names the device that was asked for.
type NoDeviceError struct {
	// Device is the id a preset carries in data.device.
	Device int
}

// Error implements the error interface.
func (e *NoDeviceError) Error() string {
	return fmt.Sprintf("no catalog is built in for device %d", e.Device)
}

// Unwrap returns ErrNoDevice so callers can match with errors.Is.
func (*NoDeviceError) Unwrap() error { return ErrNoDevice }

// BadParamError names the block, the parameter and why it was rejected.
type BadParamError struct {
	Model  string
	Key    string
	Reason string
}

// Error implements the error interface.
func (e *BadParamError) Error() string {
	return fmt.Sprintf("block %q parameter %q: %s", e.Model, e.Key, e.Reason)
}

// Unwrap returns ErrBadParam so callers can match with errors.Is.
func (*BadParamError) Unwrap() error { return ErrBadParam }
