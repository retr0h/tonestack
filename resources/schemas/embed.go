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

// Package schemas holds the contracts this project speaks, and ships them.
//
// The RigSpec schema is embedded rather than read from disk because it is
// what validates every rig: a binary that had to find its own contract on the
// filesystem could not validate anything once installed.
package schemas

import _ "embed"

// RigSpec is the contract a rig is checked against.
//
// The same file the Go types are generated from, so a constraint stated once
// is both a type and a check.
//
//go:embed rigspec.openapi.yaml
var RigSpec []byte

// CharacterTerms is the vocabulary a rig's character may use.
//
// Beside the contract rather than in it. The list will churn for months,
// every term needs a sentence of definition an OpenAPI enum has nowhere to
// put, and adding a word should be a data change rather than a schema edit
// and a regeneration.
//
//go:embed character-terms.json
var CharacterTerms []byte
