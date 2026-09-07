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
	"io"
	"os"
)

// Run builds a catalog and writes it, reporting what it did to w.
//
// This is the whole of the generate command's behaviour, so cmd/ holds only
// flags.
func Run(w io.Writer, opts Options) error {
	if opts.SchemaVersion == 0 {
		opts.SchemaVersion = defaultSchemaVersion
	}

	if opts.SourceName == "" {
		opts.SourceName = defaultSourceName
	}

	c, err := Build(opts)
	if err != nil {
		return err
	}

	// A catalog holds no channels, functions or NaN floats, so encoding it
	// cannot fail. Checking would add a branch no test can reach and hide the
	// one that matters, which is the write.
	raw, _ := json.Marshal(c)

	if err := os.WriteFile(opts.OutputPath, compress(raw), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	named := 0

	for _, b := range c.Blocks {
		if b.BasedOn != "" {
			named++
		}
	}

	_, err = fmt.Fprintf(w,
		"wrote %s: %d blocks for %s from %s, %d mapped to real gear\n",
		opts.OutputPath, len(c.Blocks), opts.DeviceName, c.Source, named)
	if err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
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
