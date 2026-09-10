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

	"github.com/retr0h/tonestack/pkg/preset"
)

// Read decodes a setlist or bundle file.
//
// A .hls and a .hlb differ only in what the payload holds, so both arrive as
// a Document carrying one or more setlists and callers address a slot the
// same way either way.
func Read(r io.Reader) (*Document, error) {
	var env envelope
	if err := json.NewDecoder(r).Decode(&env); err != nil {
		return nil, fmt.Errorf("decoding setlist: %w", err)
	}

	if env.Schema != schemaSetlist && env.Schema != schemaBundle {
		return nil, &NotASetlistError{Schema: env.Schema}
	}

	raw, err := decodePayload(env)
	if err != nil {
		return nil, err
	}

	doc := &Document{Schema: env.Schema, Version: env.Version, Meta: env.Meta}

	doc.Setlists, err = decodeSetlists(env.Schema, raw)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// decodePayload un-base64s, decompresses, and checks the result against the
// checksum the file states for it.
//
// The check is worth making: a truncated download decompresses to something
// that fails to parse thousands of lines later, and the error it produces
// names a JSON offset rather than the actual problem.
func decodePayload(env envelope) ([]byte, error) {
	packed, err := base64.StdEncoding.DecodeString(env.EncodedData)
	if err != nil {
		return nil, fmt.Errorf("decoding encoded_data: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(packed))
	if err != nil {
		return nil, fmt.Errorf("opening compressed payload: %w", err)
	}
	defer func() { _ = zr.Close() }()

	// Bounded by what the file says it holds, plus one byte so an overrun is
	// still visible to the size check below. A stream that decides for itself
	// how much memory to take is one somebody else wrote.
	raw, err := io.ReadAll(
		io.LimitReader(zr, int64(env.Compression.DecompressedSize)+1))
	if err != nil {
		return nil, fmt.Errorf("decompressing payload: %w", err)
	}

	if n := len(raw); n != env.Compression.DecompressedSize {
		return nil, &CorruptError{
			Field: "decompressed size",
			Want:  uint64(env.Compression.DecompressedSize),
			Got:   uint64(n),
		}
	}

	if sum := crc32.ChecksumIEEE(raw); sum != env.Compression.CRC32 {
		return nil, &CorruptError{
			Field: "checksum",
			Want:  uint64(env.Compression.CRC32),
			Got:   uint64(sum),
		}
	}

	return raw, nil
}

// decodeSetlists parses the payload according to the schema that wrapped it.
func decodeSetlists(schema string, raw []byte) ([]Setlist, error) {
	if schema == schemaBundle {
		var p payloadBundle
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("decoding bundle payload: %w", err)
		}

		out := make([]Setlist, 0, len(p.Setlists))
		for _, s := range p.Setlists {
			out = append(out, Setlist{Meta: s.Meta, Slots: s.Presets})
		}

		return out, nil
	}

	var p payloadSetlist
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("decoding setlist payload: %w", err)
	}

	return []Setlist{{Meta: p.Meta, Slots: p.Presets}}, nil
}

// Slot returns the preset at an address.
//
// Slots are addressed the way the device numbers them, from zero.
func (d *Document) Slot(setlist, slot int) (*preset.Data, error) {
	if setlist < 0 || setlist >= len(d.Setlists) {
		return nil, &NoSuchSlotError{
			Setlist: setlist, Slot: slot, Have: 0,
		}
	}

	s := d.Setlists[setlist]
	if slot < 0 || slot >= len(s.Slots) {
		return nil, &NoSuchSlotError{
			Setlist: setlist, Slot: slot, Have: len(s.Slots),
		}
	}

	return &s.Slots[slot], nil
}
