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

package wire

import _ "embed"

// A preset is nine sections and only two of them are a chain.
//
// Sections 4 through 7 hold controller assignments, the preset's tempo and
// settings, and the firmware that wrote it. Nothing here decodes them, and
// only sections 1 and 2 are the same in every capture, so there is nothing to
// hard-code either. Roughly 1,600 of a preset's 2,600 bytes are ones this
// project cannot generate.
//
// So a preset is written into a real one rather than assembled. This is an
// unused slot off a real HX Stomp, which is what the device itself produces
// for an empty position, and every byte outside the chain travels with it.
//
// pkg/preset does the same thing for a .hlx and for the same reason.
//
//go:embed data/hx-stomp.blank.bin
var blank []byte

// Blank returns an empty preset with everything a device expects around a
// chain.
//
// Each call returns its own document, because the caller is about to write
// into it.
func Blank() (*Document, error) { return DecodeDocument(blank) }
