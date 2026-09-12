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

// Package corpusgen measures a body of presets into statistics.
//
// It runs when the corpus changes, not on every build. The result is
// committed and embedded, so nobody needs the 67MB of presets to use what was
// learned from them. See docs/knowledge.md.
package corpusgen

// Options says what to measure and where to put the result.
type Options struct {
	// CorpusDir is the directory holding presets, searched recursively.
	CorpusDir string
	// CatalogPath is the catalog naming the models to measure. Empty means
	// the one built into this binary.
	CatalogPath string
	// OutputPath is where the gzipped statistics are written.
	OutputPath string
	// MinSamples is how many values a parameter needs before its
	// distribution is recorded.
	//
	// A median over two presets is not a measurement, it is an anecdote with
	// a decimal point.
	MinSamples int
}

// sample accumulates the values seen for one parameter before they are
// reduced to quartiles.
type sample []float64

// counter tallies how often a category appears and where it sits relative to
// the amp.
type counter struct {
	chains int
	before int
	after  int
}
