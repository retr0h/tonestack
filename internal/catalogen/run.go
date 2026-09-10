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
package catalogen

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
)

// Run builds a catalog and writes it, reporting what it did to w.
//
// This is the whole of the generate command's behaviour, so cmd/ holds only
// flags.
// Result is what a generation run produced.
type Result struct {
	// Path is the catalog that was written.
	Path string
	// Device is the hardware it describes.
	Device string
	// Source says which release it was extracted from. A catalog is only
	// true of the one it came from, so it says which.
	Source string
	// Blocks is how many the device has.
	Blocks int
	// Named is how many of those map to real-world gear. The rest are
	// modelled but unattributed, and the gap is the work left.
	Named int
}

func Run(opts Options) (Result, error) {
	if opts.SchemaVersion == 0 {
		opts.SchemaVersion = defaultSchemaVersion
	}

	if opts.SourceName == "" {
		opts.SourceName = defaultSourceName
	}

	c, err := Build(opts)
	if err != nil {
		return Result{}, err
	}

	// A catalog holds no channels, functions or NaN floats, so encoding it
	// cannot fail. Checking would add a branch no test can reach and hide the
	// one that matters, which is the write.
	raw, _ := json.Marshal(c)

	if err := os.WriteFile(opts.OutputPath, compress(raw), 0o600); err != nil {
		return Result{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	named := 0

	for _, b := range c.Blocks {
		if b.BasedOn != "" {
			named++
		}
	}

	return Result{
		Path:   opts.OutputPath,
		Device: opts.DeviceName,
		Source: c.Source,
		Blocks: len(c.Blocks),
		Named:  named,
	}, nil
}

// compress gzips the catalog.
//
// It is embedded in the binary, and a catalog is repetitive JSON: gzip takes
// 1.5MB to about 65KB, which is the difference between the device knowledge
// being worth shipping and not.
func compress(raw []byte) []byte {
	var buf bytes.Buffer

	// Compressing into a buffer cannot fail.
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(raw)
	_ = zw.Close()

	return buf.Bytes()
}
