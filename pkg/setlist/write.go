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

package setlist

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
)

// Write encodes a setlist or bundle file.
//
// The checksum and size in the envelope describe the payload, so both are
// recomputed here rather than carried over from the file that was read. A
// stale checksum would be rejected by whatever loads the file next, and the
// message it gave would blame the wrong thing.
func Write(w io.Writer, d *Document) error {
	raw, err := encodePayload(d)
	if err != nil {
		return err
	}

	var packed bytes.Buffer

	// Compressing into a buffer cannot fail, so neither call is checked. A
	// branch no test can reach is worse than none.
	zw := zlib.NewWriter(&packed)
	_, _ = zw.Write(raw)
	_ = zw.Close()

	env := envelope{
		Schema:   d.Schema,
		Version:  d.Version,
		Meta:     d.Meta,
		Encoding: Encoding,
		Compression: compression{
			Type:             CompressionZlib,
			CRC32:            crc32.ChecksumIEEE(raw),
			DecompressedSize: len(raw),
		},
		EncodedData: base64.StdEncoding.EncodeToString(packed.Bytes()),
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("encoding setlist: %w", err)
	}

	return nil
}

// encodePayload renders the inner JSON in the shape the schema calls for.
func encodePayload(d *Document) ([]byte, error) {
	if d.Schema == SchemaBundle {
		p := payloadBundle{Setlists: make([]payloadSetlist, 0, len(d.Setlists))}
		for _, s := range d.Setlists {
			p.Setlists = append(p.Setlists, payloadSetlist{Meta: s.Meta, Presets: s.Slots})
		}

		return marshalPayload(p)
	}

	if len(d.Setlists) != 1 {
		return nil, fmt.Errorf(
			"a %s file holds one setlist, not %d", SchemaSetlist, len(d.Setlists),
		)
	}

	s := d.Setlists[0]

	return marshalPayload(payloadSetlist{Meta: s.Meta, Presets: s.Slots})
}

// marshalPayload renders the payload compactly, the way a device writes it.
func marshalPayload(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encoding payload: %w", err)
	}

	return raw, nil
}
