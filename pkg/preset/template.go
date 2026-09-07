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
	"bytes"
	_ "embed"
	"fmt"
)

// blank is an untouched preset, as the device itself wrote it.
//
// A preset holds more than a signal chain: the inputs, outputs, split and
// join a device expects, its snapshots, its global and Variax settings. None
// of that is anything a person chooses, and 98.6% of real presets carry it
// while a preset built from nothing carries none.
//
// So a generated preset is written into this rather than assembled. It is a
// real slot off a real HX Stomp with no blocks in it, which is exactly what
// the device produces for an empty position.
//
//go:embed data/hx-stomp.template.hlx
var blank []byte

// Blank returns an empty preset with everything a device expects around a
// chain.
//
// Each call returns its own copy, because the caller is about to write into
// it.
func Blank() (*Document, error) { return decodeBlank(blank) }

// decodeBlank reads a template from bytes.
//
// Separate from Blank so a damaged one can be exercised. The embedded copy is
// a compile-time constant and cannot be broken at run time, but a build that
// shipped a truncated one should say so rather than produce half a preset.
func decodeBlank(raw []byte) (*Document, error) {
	doc, err := Read(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("reading the blank preset: %w", err)
	}

	return doc, nil
}
