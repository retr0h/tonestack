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
package cli

import (
	"errors"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// Hint adds the command to run next to an error somebody at a terminal can act
// on.
//
// The SDK says what went wrong and leaves the next step to whoever called it,
// because an agent over MCP calls tools rather than commands. Any other error
// comes back as it was.
func Hint(
	err error,
) error {
	switch {
	case errors.Is(err, sdk.ErrNoSuchBlock):
		return fmt.Errorf("%w, try 'tonestack catalog list'", err)
	case errors.Is(err, sdk.ErrNoSuchRecipe):
		return fmt.Errorf("%w, try 'tonestack recipes list'", err)
	default:
		return err
	}
}
