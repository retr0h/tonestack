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

package fileslots

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Catalogs hands over the catalog model names are read out of. The sdk Client
// satisfies it, and keeps the catalog it opened.
type Catalogs interface {
	// Catalog returns the catalog, opening it on first use.
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}

// Compiler reads a preset into a rig.
//
// One of the four methods pkg/compile carries, because lifting what a slot
// holds is all this package does with it.
type Compiler interface {
	// Lift reads a preset into a rig.
	Lift(doc *preset.Document, cat *catalog.Catalog) (rig.Spec, error)
}

// Flows are the operations on a slot of a file, and what they were configured
// with.
//
// Built once by whoever owns the configuration, which is the sdk Client. Each
// flow is a method taking only what differs between two calls: a path, an
// address and, for an edit, where the result goes.
//
// Every collaborator is optional: a nil one reaches the real thing, so a test
// names only what it stands something else in for. This is the shape net/http
// gives a Client, whose nil Transport means the default one.
type Flows struct {
	// Catalogs hands over the catalog. Nil reads the one built into this
	// binary.
	Catalogs Catalogs
	// Compiler moves a preset into a rig. Nil uses pkg/compile.
	Compiler Compiler
}

// catalog is the catalog these flows name gear against.
func (f *Flows) catalog(
	ctx context.Context,
) (*catalog.Catalog, error) {
	if f.Catalogs != nil {
		return f.Catalogs.Catalog(ctx)
	}

	return catalog.BuiltIn()
}

func (f *Flows) compiler() Compiler {
	if f.Compiler != nil {
		return f.Compiler
	}

	return compile.New()
}
