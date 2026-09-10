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
package preset

import (
	"errors"
	"fmt"
)

// ErrNotAPreset reports a document that is not a Line 6 preset.
var ErrNotAPreset = errors.New("not a preset")

// NotAPresetError says what the document turned out to be.
type NotAPresetError struct {
	Schema string
}

// Error implements the error interface.
func (e *NotAPresetError) Error() string {
	if e.Schema == "" {
		return "not a preset: no schema field"
	}

	return fmt.Sprintf("not a preset: schema is %q, want %q", e.Schema, Schema)
}

// Unwrap returns ErrNotAPreset so callers can match with errors.Is.
func (*NotAPresetError) Unwrap() error { return ErrNotAPreset }
