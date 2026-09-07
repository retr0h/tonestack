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

package chain

// Validate runs every layer in the order that produces the most useful first
// failure: structure, then parameters, then topology, then budget.
//
// The order is deliberate. A rig naming a model that does not exist should say
// so rather than complain that a parameter is missing from a block that was
// never found; a rig with too many blocks should say that rather than report
// the DSP overflow that follows from it.
//
// Callers wanting to know every problem at once should call the layers
// individually. This returns the first.
func Validate(l BlockLookup, s Chain, lim Limits) error {
	if err := ValidateStructure(l, s); err != nil {
		return err
	}

	if err := ValidateParams(l, s); err != nil {
		return err
	}

	if err := ValidateTopology(s, lim); err != nil {
		return err
	}

	return ValidateBudget(l, s, lim)
}
