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

package device

import (
	"errors"
	"fmt"
)

// ErrNoDevice reports that no recognised device is attached.
var ErrNoDevice = errors.New("no device found")

// ErrUnknownModel reports a Line 6 device whose product identifier this
// package does not recognise.
var ErrUnknownModel = errors.New("unrecognised model")

// UnknownModelError names the product identifier that was not recognised.
type UnknownModelError struct {
	Product uint16
}

// Error implements the error interface.
func (e *UnknownModelError) Error() string {
	return fmt.Sprintf("unrecognised model, usb product 0x%04x", e.Product)
}

// Unwrap returns ErrUnknownModel so callers can match with errors.Is.
func (*UnknownModelError) Unwrap() error { return ErrUnknownModel }

// ErrNotAPreset reports a device answering a read with something other than a
// preset document.
var ErrNotAPreset = errors.New("the device did not answer with a preset")

// NotAPresetError carries what arrived instead.
//
// The reply is kept rather than described, because a device answering
// something nobody expected is the one case worth looking at whole: this is
// how a protocol change becomes visible, and `any` is the honest type for a
// value nothing could interpret.
type NotAPresetError struct {
	// Result is what the device sent.
	Result any
}

// Shape describes the answer in whatever detail can be had.
func (e *NotAPresetError) Shape() string {
	switch v := e.Result.(type) {
	case map[any]any:
		return fmt.Sprintf("map with %d keys", len(v))
	case []byte:
		return fmt.Sprintf("%d bytes", len(v))
	default:
		return fmt.Sprintf("%T", e.Result)
	}
}

// Error implements the error interface.
func (e *NotAPresetError) Error() string {
	return fmt.Sprintf(
		"the device did not answer with a preset, but with %s", e.Shape())
}

// Unwrap returns ErrNotAPreset so callers can match with errors.Is.
func (*NotAPresetError) Unwrap() error { return ErrNotAPreset }

// document reads the preset out of a reply.
//
// A device answers an empty slot with nothing at all, which is a slot holding
// no preset rather than a failure, so that comes back as no bytes and no
// error. Anything else that is not a document is a failure, and saying so
// here — where the wire format is known — keeps a protocol change from
// reading as an empty slot at every call site.
func document(result any) ([]byte, error) {
	if result == nil {
		return nil, nil
	}

	// A preset arrives as an opaque run of bytes that MessagePack's string
	// type happens to carry.
	body, ok := result.(string)
	if !ok {
		return nil, &NotAPresetError{Result: result}
	}

	return []byte(body), nil
}
