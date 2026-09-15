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

package corpusgen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
)

// Refreshed says what refreshing the corpus statistics did.
type Refreshed struct {
	// Path is the statistics file.
	Path string
	// Skipped says which input this machine does not have, when nothing ran.
	Skipped string
	// Changed says the file was written. False when what was measured matched
	// what was already there, byte for byte.
	Changed bool
	// Stats are the measurements, so nothing has to read the file back.
	Stats *corpus.Stats
}

// Refresh measures the corpus and writes the statistics only when they differ
// from the ones already there.
//
// Run by go generate on every machine. The presets are other people's and are
// not committed, so a machine without them skips, and a machine with them
// leaves the committed file alone unless the corpus or the catalog changed.
//
// The directory itself is committed, holding the script that fetches the
// presets and the list of where they come from. So an empty one is the usual
// case on a fresh checkout, and skips the same as a missing one.
func Refresh(
	opts Options,
) (Refreshed, error) {
	skipped := Refreshed{
		Path:    opts.OutputPath,
		Skipped: "no corpus presets in " + opts.CorpusDir + ", run resources/schemas/corpus/fetch.sh",
	}

	if _, err := os.Stat(opts.CorpusDir); err != nil {
		return skipped, nil
	}

	if opts.MinSamples == 0 {
		opts.MinSamples = defaultMinSamples
	}

	cat, err := catalog.Open(opts.CatalogPath)
	if err != nil {
		return Refreshed{}, err
	}

	stats, err := Measure(opts, cat)
	if errors.Is(err, ErrNoPresets) {
		return skipped, nil
	}

	if err != nil {
		return Refreshed{}, err
	}

	// Statistics hold only numbers and strings, so encoding cannot fail.
	raw, _ := json.Marshal(stats)
	body := compress(raw)

	out := Refreshed{Path: opts.OutputPath, Stats: stats}

	if was, err := os.ReadFile(opts.OutputPath); err == nil && bytes.Equal(was, body) {
		return out, nil
	}

	if err := os.WriteFile(opts.OutputPath, body, 0o600); err != nil {
		return Refreshed{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	out.Changed = true

	return out, nil
}
