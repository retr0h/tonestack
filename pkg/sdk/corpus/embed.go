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

package corpus

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// builtIn is the measured corpus for the device this tool targets.
//
// It ships in the binary for the same reason the catalog does: measuring 4,426
// presets takes a corpus nobody wants to download, and the result is small.
//
//go:embed data/hx-stomp.stats.json.gz
var builtIn []byte

// BuiltIn returns the statistics compiled into this binary.
func BuiltIn() (*Stats, error) { return decode(builtIn) }

// decode reads statistics from bytes.
//
// Separate from BuiltIn so a corrupted archive can be exercised. The embedded
// copy is a compile-time constant and cannot be damaged at run time, but a
// build that shipped a truncated one should say so rather than panic.
func decode(packed []byte) (*Stats, error) { return Load(bytes.NewReader(packed)) }

// gzipMagic is the two bytes every gzip stream begins with.
var gzipMagic = []byte{0x1f, 0x8b}

// Load reads statistics, compressed or not.
//
// The generator writes gzip because the file is embedded, but somebody
// inspecting or hand-editing a copy will have plain JSON. Refusing one of the
// two would be an arbitrary distinction, so the stream says which it is.
func Load(r io.Reader) (*Stats, error) {
	br := bufio.NewReader(r)

	head, err := br.Peek(len(gzipMagic))
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("reading corpus statistics: %w", err)
	}

	var src io.Reader = br

	if bytes.Equal(head, gzipMagic) {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("opening corpus statistics: %w", err)
		}

		defer func() { _ = zr.Close() }()

		src = zr
	}

	var s Stats
	if err := json.NewDecoder(src).Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding corpus statistics: %w", err)
	}

	return &s, nil
}

// Param returns the distribution of one model's parameter, and whether the
// corpus saw it often enough to say anything.
func (s *Stats) Param(id catalog.ModelID, key string) (ParamStats, bool) {
	m, ok := s.Models[id]
	if !ok {
		return ParamStats{}, false
	}

	p, ok := m.Params[key]

	return p, ok
}
