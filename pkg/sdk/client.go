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
	"io"
	"sync"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/attached"
	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
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
//
// A Client is safe for concurrent use.
type Client struct {
	opts options

	// mu guards cat, the catalog opened on first use.
	mu  sync.Mutex
	cat *catalog.Catalog

	// claim is held by the Session this Client has open. One slot: a
	// second Open waits for it.
	claim chan struct{}
}

// options are what New was given.
//
// Unexported, with a function per setting, so adding one changes no call
// already written.
type options struct {
	// catalog is a generated catalog. Empty means the built-in one.
	catalog string
	// stats is measured corpus statistics. Empty means the built-in ones.
	stats string
	// recipes is a directory of rigs. Empty means the ones that ship.
	recipes string
	// backupDir is where a slot's old contents go. Empty means the state
	// directory.
	backupDir string
	// capture receives each device answer a read gets. Nil keeps nothing.
	capture io.Writer
	// trace receives every USB frame in and out. Nil traces nothing.
	trace io.Writer
	// devices is the bus. Nil until New fills in USB.
	devices device.Opener
}

// Option changes how a Client works.
type Option func(*options)

// WithCatalog reads a generated catalog instead of the built-in one.
func WithCatalog(
	path string,
) Option {
	return func(o *options) { o.catalog = path }
}

// WithStats reads corpus statistics instead of the built-in ones.
func WithStats(
	path string,
) Option {
	return func(o *options) { o.stats = path }
}

// WithRecipes reads rigs from a directory instead of the ones that ship.
//
// Scaffold writes there too, and needs it: a new rig is not written into
// wherever the program happened to run.
func WithRecipes(
	dir string,
) Option {
	return func(o *options) { o.recipes = dir }
}

// WithBackupDir is where a slot's old contents go before a device write.
//
// Without it they go to $XDG_STATE_HOME/tonestack/presets, or to
// ~/.local/state/tonestack/presets when that variable is unset.
func WithBackupDir(
	dir string,
) Option {
	return func(o *options) { o.backupDir = dir }
}

// WithCapture receives each device answer a read gets, verbatim.
//
// It is how the wire format was read in the first place: a preset arrives as
// the bytes the device sent, and an answer nothing here decodes arrives as
// JSON.
func WithCapture(
	w io.Writer,
) Option {
	return func(o *options) { o.capture = w }
}

// WithTrace receives every USB frame in and out.
func WithTrace(
	w io.Writer,
) Option {
	return func(o *options) { o.trace = w }
}

// New builds a Client.
//
// Usable with no options at all, because the common case is a caller who
// wants the built-in catalog, the built-in statistics, the rigs that ship and
// whatever device happens to be plugged in. New opens nothing and cannot fail.
func New(
	opts ...Option,
) *Client {
	var o options

	for _, fn := range opts {
		fn(&o)
	}

	if o.devices == nil {
		o.devices = device.NewUSB(o.trace)
	}

	return &Client{opts: o, claim: make(chan struct{}, 1)}
}

// Catalog is the catalog this Client names gear against.
//
// Opened on first use and kept, so a renderer reads the same catalog the
// operation did. A catalog that would not open is not kept, and the next call
// tries again.
func (c *Client) Catalog(
	ctx context.Context,
) (*catalog.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cat != nil {
		return c.cat, nil
	}

	cat, err := catalog.Open(c.opts.catalog)
	if err != nil {
		return nil, err
	}

	c.cat = cat

	return cat, nil
}

// catalogs hands a flow the Client's own catalog.
//
// The flows ask for a catalog by path. The Client already knows which one it
// was given, so the path is ignored and every call reads the one opened once.
//
// It holds ctx because slots.Catalogs and presets.Catalogs take none; chunk
// 49.6 replaces both with a catalog field on the flows, and this goes with them.
type catalogs struct {
	client *Client
	ctx    context.Context
}

// Open returns the Client's catalog.
func (k catalogs) Open(
	_ string,
) (*catalog.Catalog, error) {
	return k.client.Catalog(k.ctx)
}

// Devices reports the hardware attached to this machine.
//
// Recognised hardware only: a bus holds keyboards and webcams, and a list of
// those is not an answer to "what can I write a preset to".
func (c *Client) Devices(
	ctx context.Context,
) (Attached, error) {
	return attached.ListWith(ctx, c.opts.devices)
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
func (c *Client) Blocks(
	ctx context.Context,
	f Filter,
) (Blocks, error) {
	cat, err := c.Catalog(ctx)
	if err != nil {
		return Blocks{}, err
	}

	return catalogview.List(cat, catalogview.Filter(f)), nil
}

// Block reports one block and everything it accepts.
func (c *Client) Block(
	ctx context.Context,
	id string,
) (catalog.Block, error) {
	cat, err := c.Catalog(ctx)
	if err != nil {
		return catalog.Block{}, err
	}

	return catalogview.Show(cat, id)
}

// Corpus says what to read out of the measurements.
type Corpus struct {
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
func (c *Client) Measurements(
	ctx context.Context,
	in Corpus,
) (Measured, error) {
	if err := ctx.Err(); err != nil {
		return Measured{}, err
	}

	return corpusview.Show(ctx, corpusview.Options{
		StatsPath:  c.opts.stats,
		Catalogs:   c,
		Model:      in.Model,
		Instrument: in.Instrument,
	})
}

// Recipes reads every rig this Client was given.
func (c *Client) Recipes(
	ctx context.Context,
) (Recipes, error) {
	if err := ctx.Err(); err != nil {
		return Recipes{}, err
	}

	return recipes.List(c.opts.recipes)
}

// Recipe reads one rig, and what the rest of the set says about it.
func (c *Client) Recipe(
	ctx context.Context,
	id string,
) (Recipe, error) {
	if err := ctx.Err(); err != nil {
		return Recipe{}, err
	}

	return recipes.Show(c.opts.recipes, id)
}

// NewRecipe describes the rig to scaffold.
type NewRecipe struct {
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
//
// The rig is written into the directory WithRecipes named. A Client given
// none is refused rather than writing wherever the program happened to run.
func (c *Client) Scaffold(
	ctx context.Context,
	in NewRecipe,
) (Scaffolded, error) {
	if err := ctx.Err(); err != nil {
		return Scaffolded{}, err
	}

	return recipes.New(ctx, recipes.NewOptions{
		Dir:        c.opts.recipes,
		ID:         in.ID,
		Name:       in.Name,
		Band:       in.Band,
		Instrument: in.Instrument,
		Amp:        in.Amp,
		Cab:        in.Cab,
		Pedals:     in.Pedals,
		Catalogs:   c,
		From:       in.From,
		Kind:       in.Kind,
	})
}

// Make says which rig to build and where to put it.
type Make struct {
	// RecipeID names the curated knowledge to build from.
	RecipeID string
	// OutputPath is where the preset is written.
	OutputPath string
}

// Build compiles a rig into a preset and writes it.
//
// Reporting what it chose matters as much as writing the file. A generated
// preset is a set of decisions, and a wrong amp should be visible before
// anybody plugs in rather than after.
func (c *Client) Build(
	ctx context.Context,
	in Make,
) (Made, error) {
	if err := ctx.Err(); err != nil {
		return Made{}, err
	}

	return presets.Make(presets.MakeOptions{
		Deps:        presets.Deps{Catalogs: catalogs{client: c, ctx: ctx}},
		RecipeID:    in.RecipeID,
		RecipesDir:  c.opts.recipes,
		CatalogPath: c.opts.catalog,
		StatsPath:   c.opts.stats,
		OutputPath:  in.OutputPath,
	})
}
