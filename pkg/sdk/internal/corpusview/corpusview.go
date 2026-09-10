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

// Options says what to show.
type Options struct {
	// StatsPath is measured statistics to read instead of the built-in ones.
	StatsPath string
	// CatalogPath is a catalog to read instead of the built-in one.
	CatalogPath string
	// Model shows one model's parameter distributions.
	Model string
	// Instrument shows what chains for one instrument tend to hold.
	Instrument string
}

// Show reads what the corpus says.
//
// The measurements and, when one model was asked about, the catalog beside
// them: what players chose means little without what Line 6 chose.
func Show(opts Options) (result.Measured, error) {
	stats, err := open(opts.StatsPath)
	if err != nil {
		return result.Measured{}, err
	}

	if opts.Model == "" {
		return result.Measured{Stats: stats, Instrument: opts.Instrument}, nil
	}

	id := catalog.ModelID(opts.Model)

	// Asked here rather than while drawing, so a model nobody measured is an
	// error from the operation and not a table with nothing in it.
	if _, ok := stats.Models[id]; !ok {
		return result.Measured{}, &NotMeasuredError{Model: id}
	}

	cat, err := catalog.Open(opts.CatalogPath)
	if err != nil {
		return result.Measured{}, err
	}

	return result.Measured{Stats: stats, Catalog: cat, Model: id}, nil
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

	defer func() { _ = f.Close() }()

	return corpus.Load(f)
}
