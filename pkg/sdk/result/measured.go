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

import (
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
)

// Measured is what the corpus recorded, and what was asked of it.
//
// Two questions come out of the same measurements: what players did with one
// model, and what chains of a kind are shaped like. Which was asked decides
// what there is to say, so the answer carries it rather than leaving somebody
// to infer it from which fields are set.
type Measured struct {
	// Stats are the measurements themselves.
	Stats *corpus.Stats
	// Catalog is what the device accepts, so what players chose can be read
	// beside what Line 6 chose. Nil unless a model was asked about.
	Catalog *catalog.Catalog
	// Model is the model asked about. Empty for the grammar.
	Model catalog.ModelID
	// Instrument narrows the grammar to one kind of chain. Empty for all.
	Instrument string
}

// AboutOne says whether one model was asked about, rather than the grammar.
func (m Measured) AboutOne() bool { return m.Model != "" }
