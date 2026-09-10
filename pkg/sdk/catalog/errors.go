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
)

// ErrBadParam reports a parameter that does not exist on a block, or whose
// value does not fit the declared type or range.
var ErrBadParam = errors.New("invalid parameter")

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
