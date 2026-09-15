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
	"fmt"
	"io"
	"sync"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/attached"
	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
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
// A Client is safe for concurrent use. Build one with New. The zero Client
// reads files and the built-in catalog, but has no bus to reach a device
// through.
type Client struct {
	opts options

	// fileFlows and deviceFlows are the slot operations, configured from opts
	// once, on first use, so a Client that did not come from New still has
	// them.
	flowsOnce   sync.Once
	fileFlows   *fileslots.Flows
	deviceFlows *deviceslots.Flows

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
	// userRecipes is somebody's own directory of rigs, layered over recipes.
	// Empty layers nothing.
	userRecipes string
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
// Scaffold and Extend write there too, unless WithUserRecipes names somewhere
// else, and need one or the other: a new rig is not written into wherever the
// program happened to run.
func WithRecipes(
	dir string,
) Option {
	return func(o *options) { o.recipes = dir }
}

// WithUserRecipes layers somebody's own directory of rigs over the ones that
// ship, or over the directory WithRecipes named.
//
// A rig of theirs takes the place of one beneath when the two share an
// identifier or alias, in any case, so Recipes lists the rig Recipe and Build
// find. Variants are read across both, so a rig of theirs made from a shipped
// one with Extend shows under it. A directory that is not there holds no rigs;
// one that cannot be read is an error.
//
// A file in it that is not a rig is reported by Recipes. It stops Recipe,
// Build and Extend only when it may be the rig asked for: when its filename,
// or the id or aliases it states, is the name asked for or a name of the rig
// found. Otherwise one mistake in a directory of their own would stop every
// shipped rig building, and a broken rig of theirs would never be quietly
// passed over for the shipped one it was written to replace.
//
// Scaffold and Extend write here. The library reads no environment for this:
// a program that keeps rigs under $XDG_DATA_HOME resolves that and passes the
// directory in.
func WithUserRecipes(
	dir string,
) Option {
	return func(o *options) { o.userRecipes = dir }
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

// buildFlows builds the slot flows the first time anything asks.
func (c *Client) buildFlows() {
	c.flowsOnce.Do(func() {
		// The flows read the catalog through the Client, so they share the
		// one it opens and keeps.
		c.fileFlows = &fileslots.Flows{Catalogs: c}

		devices := &deviceslots.Flows{Catalogs: c, Capture: c.opts.capture}
		devices.Backups = backup.New(c.opts.backupDir, deviceslots.NewDecoder(devices))
		c.deviceFlows = devices
	})
}

// fileOperations are the flows over a .hls, .hlb or .hlx on disk.
func (c *Client) fileOperations() *fileslots.Flows {
	c.buildFlows()

	return c.fileFlows
}

// deviceOperations are the flows over an attached device.
func (c *Client) deviceOperations() *deviceslots.Flows {
	c.buildFlows()

	return c.deviceFlows
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

// Devices reports the hardware attached to this machine.
//
// Recognised hardware only: a bus holds keyboards and webcams, and a list of
// those is not an answer to "what can I write a preset to".
func (c *Client) Devices(
	ctx context.Context,
) (Attached, error) {
	return attached.ListWith(ctx, c.opts.devices)
}

// Filter narrows what Blocks reports. All three narrow together.
type Filter struct {
	// Category keeps only blocks of one kind: amp, cab, drive.
	Category string
	// Subcategory keeps only blocks Line 6 tags this way: Guitar, Bass.
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

// corpus is what every question of the measurements reads.
func (c *Client) corpus() corpusview.Options {
	return corpusview.Options{StatsPath: c.opts.stats, Catalogs: c}
}

// ModelMeasurements reports what players did with one model: how they set
// each parameter across every measured preset that used it.
//
// A model the corpus never saw, including an empty one, is refused rather than
// answered with nothing.
func (c *Client) ModelMeasurements(
	ctx context.Context,
	model string,
) (Measured, error) {
	if err := ctx.Err(); err != nil {
		return Measured{}, err
	}

	return corpusview.Model(ctx, c.corpus(), model)
}

// ChainMeasurements reports what chains tend to hold: which kinds of block,
// and which side of the amp they sit on.
//
// instrument narrows that to guitar or bass. Empty asks about every chain.
func (c *Client) ChainMeasurements(
	ctx context.Context,
	instrument string,
) (Measured, error) {
	if err := ctx.Err(); err != nil {
		return Measured{}, err
	}

	return corpusview.Chains(c.corpus(), instrument)
}

// rigs is where this Client reads rigs from.
func (c *Client) rigs() recipes.Source {
	return recipes.Source{Dir: c.opts.recipes, User: c.opts.userRecipes}
}

// recipesHome is where Scaffold and Extend write: the directory of somebody's
// own when there is one, and the one WithRecipes named otherwise.
func (c *Client) recipesHome() string {
	if c.opts.userRecipes != "" {
		return c.opts.userRecipes
	}

	return c.opts.recipes
}

// Recipes reads every rig this Client was given.
//
// Dir is the directory WithUserRecipes named where there is one, since that
// is where somebody's own rigs are, and the one WithRecipes named otherwise.
func (c *Client) Recipes(
	ctx context.Context,
) (Recipes, error) {
	if err := ctx.Err(); err != nil {
		return Recipes{}, err
	}

	return recipes.List(c.rigs())
}

// Recipe reads one rig, and what the rest of the set says about it.
func (c *Client) Recipe(
	ctx context.Context,
	id string,
) (Recipe, error) {
	if err := ctx.Err(); err != nil {
		return Recipe{}, err
	}

	return recipes.Show(c.rigs(), id)
}

// NewRecipe describes a rig to scaffold from the gear it names.
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
}

// Scaffold writes a rig, after checking the gear it names exists.
//
// Checking first is the point. A rig naming gear no device models is only
// found out when somebody tries to build from it, and by then the name has
// usually been copied somewhere else too.
//
// The rig is written into the directory WithUserRecipes named, or failing that
// the one WithRecipes named. A Client given neither is refused rather than
// writing wherever the program happened to run.
func (c *Client) Scaffold(
	ctx context.Context,
	in NewRecipe,
) (Scaffolded, error) {
	if err := ctx.Err(); err != nil {
		return Scaffolded{}, err
	}

	return recipes.New(ctx, recipes.NewOptions{
		Dir:        c.recipesHome(),
		ID:         in.ID,
		Name:       in.Name,
		Band:       in.Band,
		Instrument: in.Instrument,
		Amp:        in.Amp,
		Cab:        in.Cab,
		Pedals:     in.Pedals,
		Catalogs:   c,
	})
}

// ExtendRecipe describes a rig to start as a copy of another.
type ExtendRecipe struct {
	// From is the rig to copy, by identifier or alias. Required.
	From string
	// ID is the new rig's identifier, and its filename stem.
	ID string
	// Name is the player or style the copy is about. Empty keeps the name
	// the copied rig has.
	Name string
	// Kind is what the copy is attributed to: artist, band, song, genre or
	// sound. Empty keeps the copied rig's.
	Kind string
}

// Extend writes a new rig as a copy of one that exists.
//
// The copy is a whole rig, comments and citations included, and records the
// rig it came from in `extends`. Nothing merges the two: editing the copy does
// not touch the original. No gear is checked, because the rig it copies
// already resolved when it was written.
//
// The rig is written where Scaffold writes. From is looked for there first,
// then in the rigs beneath: the ones WithRecipes named when WithUserRecipes
// was given too, and the ones that ship otherwise. The report's Instrument and
// Amp are the copied rig's, and so is its Name unless in.Name gave another.
func (c *Client) Extend(
	ctx context.Context,
	in ExtendRecipe,
) (Scaffolded, error) {
	if err := ctx.Err(); err != nil {
		return Scaffolded{}, err
	}

	// Without a rig to copy this would scaffold one from no gear at all,
	// which is a different operation with a different check.
	if in.From == "" {
		return Scaffolded{}, fmt.Errorf("%w: name the rig to copy", ErrNoSuchRecipe)
	}

	// A directory WithRecipes named is where the copy goes when there is no
	// directory of their own, so it cannot also be what that one sits on.
	base := ""
	if c.opts.userRecipes != "" {
		base = c.opts.recipes
	}

	return recipes.New(ctx, recipes.NewOptions{
		Dir:      c.recipesHome(),
		Base:     base,
		From:     in.From,
		ID:       in.ID,
		Name:     in.Name,
		Kind:     in.Kind,
		Catalogs: c,
	})
}

// Build compiles a shipped or configured rig into a preset, and writes it to
// out.
//
// Reporting what it chose matters as much as writing the file. A generated
// preset is a set of decisions, and a wrong amp should be visible before
// anybody plugs in rather than after.
//
// existing says what happens to a file already at out: ReplaceExisting puts
// the preset in its place, and KeepExisting refuses it with an error matching
// fs.ErrExist.
func (c *Client) Build(
	ctx context.Context,
	recipeID string,
	out string,
	existing Existing,
) (Made, error) {
	return presets.Make(ctx, presets.MakeOptions{
		Deps:       presets.Deps{Catalogs: c},
		RecipeID:   recipeID,
		Rigs:       c.rigs(),
		StatsPath:  c.opts.stats,
		OutputPath: out,
		Existing:   existing,
	})
}
