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
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/compile"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// Catalogs opens the catalog a rig is built against.
type Catalogs interface {
	// Open reads the catalog at path, or the built-in one when path is empty.
	Open(path string) (*catalog.Catalog, error)
}

// Recipes finds the curated rig a build starts from.
type Recipes interface {
	// Find returns the rig with the given identifier.
	Find(dir, id string) (riggen.RigSpec, error)
}

// Compiler turns a rig into a chain a device has room for.
//
// The other half of what pkg/compile carries, Lift and Lower, belongs to
// reading a device rather than building a preset, so it is not named here.
type Compiler interface {
	// Resolve turns a rig and a catalog into a chain.
	Resolve(
		spec riggen.RigSpec, cat *catalog.Catalog, stats *corpus.Stats,
	) (chain.Chain, []compile.Added, error)
	// Fit drops what a device has no room for.
	Fit(spec chain.Chain, cat *catalog.Catalog, lim chain.Limits) chain.Chain
}

// Deps are the collaborators building a preset works through.
//
// Every field optional: a zero value reaches the real thing, so a caller
// names only what it wants to stand something else in for.
type Deps struct {
	// Catalogs opens catalogs. Nil reads them from disk.
	Catalogs Catalogs
	// Recipes finds curated rigs. Nil reads the ones in the binary.
	Recipes Recipes
	// Compiler turns a rig into a chain. Nil uses pkg/compile.
	Compiler Compiler
}

func (d Deps) catalogs() Catalogs {
	if d.Catalogs != nil {
		return d.Catalogs
	}

	return catalog.Files{}
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
