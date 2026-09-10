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
