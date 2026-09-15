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

// Package corpusview shows what a corpus of presets was measured to say.
package corpusview

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// ErrNotMeasured reports that the corpus never saw a model.
var ErrNotMeasured = errors.New("not measured")

// NotMeasuredError names the model nothing was learned about.
type NotMeasuredError struct {
	Model catalog.ModelID
}

func (e *NotMeasuredError) Error() string {
	return fmt.Sprintf(
		"not measured: no preset in the corpus uses %s", e.Model)
}

func (*NotMeasuredError) Unwrap() error { return ErrNotMeasured }

// Options is what every question of the corpus reads.
type Options struct {
	// StatsPath is measured statistics to read instead of the built-in ones.
	StatsPath string
	// Catalogs hands over the catalog. Asked only about a model.
	Catalogs Catalogs
}

// Model reads what players did with one model.
//
// The measurements and the catalog beside them: what players chose means
// little without what Line 6 chose. A model nobody names is one nobody
// measured, and is refused the same way.
func Model(
	ctx context.Context,
	opts Options,
	model string,
) (result.Measured, error) {
	stats, err := open(opts.StatsPath)
	if err != nil {
		return result.Measured{}, err
	}

	id := catalog.ModelID(model)

	// Asked here rather than while drawing, so a model nobody measured is an
	// error from the operation and not a table with nothing in it.
	if _, ok := stats.Models[id]; !ok {
		return result.Measured{}, &NotMeasuredError{Model: id}
	}

	cat, err := opts.Catalogs.Catalog(ctx)
	if err != nil {
		return result.Measured{}, err
	}

	return result.Measured{Stats: stats, Catalog: cat, Model: id}, nil
}

// Chains reads what chains tend to hold, for one instrument or, with none
// named, for every one.
//
// No catalog is read: the grammar of a chain is block kinds, and those need no
// names.
func Chains(
	opts Options,
	instrument string,
) (result.Measured, error) {
	stats, err := open(opts.StatsPath)
	if err != nil {
		return result.Measured{}, err
	}

	return result.Measured{Stats: stats, Instrument: instrument}, nil
}

// open reads statistics, falling back to the ones in this binary.
func open(path string) (*corpus.Stats, error) {
	if path == "" {
		return corpus.BuiltIn()
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	return corpus.Load(f)
}
