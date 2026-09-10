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

package slot

// Value adapts a slot to a command-line flag, so every command that takes a
// slot takes the same two forms.
//
// It writes through to an int the options struct already holds, which keeps
// the addressing convention here rather than in each command.
type Value struct {
	target *int
}

// NewValue returns a flag writing into target.
func NewValue(target *int) *Value { return &Value{target: target} }

// Set parses a label or an index.
func (v *Value) Set(s string) error {
	n, err := parse(s)
	if err != nil {
		return err
	}

	*v.target = n

	return nil
}

// String renders the slot the way the hardware labels it.
func (v *Value) String() string {
	if v.target == nil {
		return ""
	}

	return Label(*v.target)
}

// Type names the flag's argument in help output.
func (*Value) Type() string { return "slot" }
