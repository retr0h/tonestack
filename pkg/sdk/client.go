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

package sdk

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/attached"
	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/presets"
	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
)

// Client is what a wrapper holds.
//
// One type to rally around, so a terminal, a service and a TUI reach the same
// operations the same way. Every method answers with a value and decides
// nothing about how it looks: the caller draws a table, serialises JSON or
// keeps a cursor in it, and none of those three has to know about the others.
//
// Where a device is needed the Client finds one; where it is not, the same
// Client works with no hardware attached. That is what lets a service compile
// rigs on a machine that has never seen a Helix.
type Client struct {
	opts Options
}

// Options are what New was given.
//
// Empty for now. It is here so that adding the first real one — a catalog
// somewhere else, a device chosen rather than found — does not change the
// shape of every call already written against this.
type Options struct{}

// Option changes how a Client works.
type Option func(*Options)

// New builds a Client.
//
// Usable with no options at all, because the common case is a caller who
// wants the built-in catalog, the built-in statistics and whatever device
// happens to be plugged in.
func New(opts ...Option) *Client {
	var o Options

	for _, fn := range opts {
		fn(&o)
	}

	return &Client{opts: o}
}

// Devices reports the hardware attached to this machine.
//
// Recognised hardware only: a bus holds keyboards and webcams, and a list of
// those is not an answer to "what can I write a preset to".
func (c *Client) Devices(ctx context.Context) (Attached, error) {
	return attached.List(ctx)
}

// Filter narrows what Blocks reports.
type Filter struct {
	// Category keeps only blocks of one kind — amp, cab, drive.
	Category string
	// Subcategory keeps only blocks Line 6 tags this way — Guitar, Bass.
	Subcategory string
	// Search keeps only blocks whose name or real-world gear mentions this.
	Search string
}

// Blocks reports what a device can do, narrowed to what was asked for.
//
// An empty CatalogPath means the catalog built into this binary, which is
// what anyone who has not generated their own wants.
func (c *Client) Blocks(catalogPath string, f Filter) (Blocks, error) {
	return catalogview.List(catalogPath, catalogview.Filter(f))
}

// Block reports one block and everything it accepts.
func (c *Client) Block(catalogPath, id string) (catalog.Block, error) {
	return catalogview.Show(catalogPath, id)
}

// Corpus says what to read out of the measurements.
type Corpus struct {
	// StatsPath is measured statistics to read instead of the built-in ones.
	StatsPath string
	// CatalogPath is a catalog to read instead of the built-in one.
	CatalogPath string
	// Model asks for one model's parameter distributions.
	Model string
	// Instrument asks what chains for one instrument tend to hold.
	Instrument string
}

// Measurements reports what the corpus recorded.
//
// Two questions come out of the same measurements: what players did with one
// model, and what chains of a kind are shaped like. Naming a Model asks the
// first; leaving it empty asks the second.
func (c *Client) Measurements(in Corpus) (Measured, error) {
	return corpusview.Show(corpusview.Options(in))
}

// Recipes reads every rig under a directory.
//
// An empty dir means the rigs that ship with this library, which is the case
// for anyone who has not written their own.
func (c *Client) Recipes(dir string) (Recipes, error) {
	return recipes.List(dir)
}

// Recipe reads one rig, and what the rest of the set says about it.
func (c *Client) Recipe(dir, id string) (Recipe, error) {
	return recipes.Show(dir, id)
}

// NewRecipe describes the rig to scaffold.
type NewRecipe struct {
	// Dir is where recipes live. Empty writes beside the ones that ship,
	// which is not usually what anybody wants.
	Dir string
	// ID is the identifier, and the filename stem.
	ID string
	// Name is the player or style, as a person would write it.
	Name string
	// Band is the group, where there is one.
	Band string
	// Instrument is guitar or bass.
	Instrument string
	// Amp is the real-world amplifier. Required: it is the one thing nothing
	// downstream recovers from getting wrong.
	Amp string
	// Cab is the real-world cabinet. Empty takes the amp's own pairing.
	Cab string
	// Pedals are real-world pedals, in signal order.
	Pedals []string
	// CatalogPath is a catalog to check against instead of the built-in one.
	CatalogPath string
	// From is a rig to copy, by identifier. The copy is a whole rig and
	// records where it came from in `extends`; nothing merges the two.
	From string
	// Kind is what the new rig is attributed to: artist, band, song, genre
	// or sound. Only read when copying, since a scaffold from nothing is an
	// artist.
	Kind string
}

// Scaffold writes a rig, after checking the gear it names exists.
//
// Checking first is the point. A rig naming gear no device models is only
// found out when somebody tries to build from it, and by then the name has
// usually been copied somewhere else too.
func (c *Client) Scaffold(in NewRecipe) (Scaffolded, error) {
	return recipes.New(recipes.NewOptions(in))
}

// Make says which rig to build and where to put it.
type Make struct {
	// RecipeID names the curated knowledge to build from.
	RecipeID string
	// RecipesDir is where recipes live. Empty means the ones that ship.
	RecipesDir string
	// CatalogPath is the generated catalog for the target device. Empty
	// means the one built into this binary.
	CatalogPath string
	// StatsPath is measured corpus statistics. Empty means the ones built
	// into this binary.
	StatsPath string
	// OutputPath is where the preset is written.
	OutputPath string
}

// Build compiles a rig into a preset and writes it.
//
// Reporting what it chose matters as much as writing the file. A generated
// preset is a set of decisions, and a wrong amp should be visible before
// anybody plugs in rather than after.
func (c *Client) Build(in Make) (Made, error) {
	return presets.Make(presets.MakeOptions{
		RecipeID:    in.RecipeID,
		RecipesDir:  in.RecipesDir,
		CatalogPath: in.CatalogPath,
		StatsPath:   in.StatsPath,
		OutputPath:  in.OutputPath,
	})
}
