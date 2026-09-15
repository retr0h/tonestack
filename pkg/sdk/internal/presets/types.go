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

package presets

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Catalogs hands over the catalog a rig is built against. The sdk Client
// satisfies it, and keeps the catalog it opened.
type Catalogs interface {
	// Catalog returns the catalog, opening it on first use.
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}

// Recipes finds the curated rig a build starts from.
type Recipes interface {
	// Find returns the rig with the given identifier, from where src says.
	Find(src recipes.Source, id string) (rig.Spec, error)
}

// Compiler turns a rig into a preset a device has room for.
//
// Three of the four methods pkg/compile carries. Resolve and Fit build a chain
// from a recipe, and Lower writes a rig into a preset. Lift reads a slot
// rather than building one, so it is not named here.
type Compiler interface {
	// Resolve turns a rig and a catalog into a chain.
	Resolve(
		spec rig.Spec, cat *catalog.Catalog, stats *corpus.Stats,
	) (chain.Chain, []compile.Added, []compile.Moved, error)
	// Fit drops what a device has no room for.
	Fit(spec chain.Chain, cat *catalog.Catalog, lim chain.Limits) chain.Chain
	// Lower writes a rig into a preset.
	Lower(doc *preset.Document, spec rig.Spec, cat *catalog.Catalog) error
}

// Deps are the collaborators building a preset works through.
//
// Every field optional: a zero value reaches the real thing, so a caller
// names only what it wants to stand something else in for.
type Deps struct {
	// Catalogs hands over the catalog. Nil reads the one built into this
	// binary.
	Catalogs Catalogs
	// Recipes finds curated rigs. Nil reads the ones in the binary.
	Recipes Recipes
	// Compiler turns a rig into a chain. Nil uses pkg/compile.
	Compiler Compiler
}

func (d Deps) catalog(
	ctx context.Context,
) (*catalog.Catalog, error) {
	if d.Catalogs != nil {
		return d.Catalogs.Catalog(ctx)
	}

	return catalog.BuiltIn()
}

func (d Deps) recipes() Recipes {
	if d.Recipes != nil {
		return d.Recipes
	}

	return recipes.Store{}
}

func (d Deps) compiler() Compiler {
	if d.Compiler != nil {
		return d.Compiler
	}

	return compile.New()
}
