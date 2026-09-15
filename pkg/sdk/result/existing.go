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

// Existing is what a write does about a file already at the path it was
// given: ReplaceExisting or KeepExisting.
//
// It travels beside the output path, because it is a decision about that path
// and nothing else. A caller who looks first and writes second leaves a gap
// another program can create the file in; one who passes KeepExisting has the
// write itself refuse the file, whenever it appeared.
type Existing int

// What a write does about a file already there.
const (
	// ReplaceExisting puts the new file in its place, whole, in one step. The
	// zero value, so a caller who does not say gets what a write has always
	// done.
	ReplaceExisting Existing = iota
	// KeepExisting leaves the file alone, and the write fails with an error
	// matching fs.ErrExist. No other failure matches it.
	KeepExisting
)
