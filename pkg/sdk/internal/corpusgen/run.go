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
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// defaultMinSamples is how many values a parameter needs before its
// distribution is worth recording.
const defaultMinSamples = 5

// Run measures a corpus and writes the statistics, reporting what it found.

func Run(opts Options) (result.Counted, error) {
	if opts.MinSamples == 0 {
		opts.MinSamples = defaultMinSamples
	}

	cat, err := catalog.Open(opts.CatalogPath)
	if err != nil {
		return result.Counted{}, err
	}

	stats, err := Measure(opts, cat)
	if err != nil {
		return result.Counted{}, err
	}

	// Statistics hold only numbers and strings, so encoding cannot fail.
	raw, _ := json.Marshal(stats)

	if err := os.WriteFile(opts.OutputPath, compress(raw), 0o600); err != nil {
		return result.Counted{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	return result.Counted{Path: opts.OutputPath, Stats: stats}, nil
}

// compress gzips the statistics, which are repetitive JSON and embedded in
// the binary.
func compress(raw []byte) []byte {
	var buf bytes.Buffer

	// Compressing into a buffer cannot fail.
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(raw)
	_ = zw.Close()

	return buf.Bytes()
}
