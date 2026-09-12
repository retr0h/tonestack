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

package compile

import (
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	riggen "github.com/retr0h/tonestack/pkg/sdk/internal/gen"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// Compiler is this package's work as a value.
//
// It holds nothing, and every method is the package-level function of the
// same name. The type exists so that a caller can say what it depends on and
// stand something else in its place, which a package-level function does not
// allow. The pair is the one net/http draws between Get and a Client.
//
// The interface a caller needs is the caller's to declare, and will be
// smaller than this: reading a device wants Lift and Lower, building a rig
// wants Resolve and Fit, and neither wants all four.
type Compiler struct{}

// New returns a Compiler.
func New() *Compiler { return &Compiler{} }

// Lift reads a preset into a rig.
func (*Compiler) Lift(
	doc *preset.Document,
	cat *catalog.Catalog,
) (riggen.RigSpec, error) {
	return Lift(doc, cat)
}

// Lower writes a rig back into a preset.
func (*Compiler) Lower(
	doc *preset.Document,
	spec riggen.RigSpec,
	cat *catalog.Catalog,
) error {
	return Lower(doc, spec, cat)
}

// Resolve turns a rig and a catalog into a chain.
func (*Compiler) Resolve(
	spec riggen.RigSpec,
	cat *catalog.Catalog,
	stats *corpus.Stats,
) (chain.Chain, []Added, []Moved, error) {
	return Resolve(spec, cat, stats)
}

// Fit drops what a device has no room for.
func (*Compiler) Fit(
	spec chain.Chain,
	cat *catalog.Catalog,
	lim chain.Limits,
) chain.Chain {
	return Fit(spec, cat, lim)
}
