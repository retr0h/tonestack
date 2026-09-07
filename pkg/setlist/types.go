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
// Package setlist reads and writes Line 6 setlist and bundle files.
//
// Both are the same envelope: a small JSON wrapper whose encoded_data field
// holds zlib-compressed, base64-encoded JSON. What is inside differs.
//
//	.hls  L6Setlist        one setlist — 128 slots
//	.hlb  L6PresetBundle   every setlist on the device — 8 x 128 slots
//
// A bundle is what HX Edit writes when it backs a device up, which makes it
// the only file that says what the whole device currently holds.
//
// Neither format is published. Everything here was established by reading
// real files. See docs/preset-format.md.
package setlist

import (
	"encoding/json"

	"github.com/retr0h/tonestack/pkg/preset"
)

// Schemas this package reads.
const (
	// SchemaSetlist is one setlist.
	SchemaSetlist = "L6Setlist"
	// SchemaBundle is every setlist on a device.
	SchemaBundle = "L6PresetBundle"
)

// Encoding is the only encoding seen in the wild.
const Encoding = "Base64"

// CompressionZlib is the only compression seen in the wild.
const CompressionZlib = "zlib"

// SlotsPerSetlist is how many slots a setlist holds.
//
// Every device-written file holds exactly this many, including the empty
// ones. A slot is a position, not a preset, so a setlist with a gap in the
// middle still has 128 entries.
const SlotsPerSetlist = 128

// Document is a setlist or bundle file, decoded.
//
// The envelope is kept as it was read so a file written back differs only
// where a slot changed. Meta is raw for the same reason: it carries fields
// this package has no opinion about and would otherwise drop.
type Document struct {
	// Schema decides which shape Write emits.
	Schema string
	// Version is the schema version the file declared.
	Version int
	// Meta is the envelope's metadata, preserved verbatim.
	Meta json.RawMessage
	// Setlists holds one entry for a setlist file, several for a bundle.
	Setlists []Setlist
}

// Address names one slot.
//
// A setlist file has a single setlist, so Setlist is zero for those and only
// matters for a bundle, where it selects which of the device's setlists the
// slot belongs to.
type Address struct {
	Setlist int
	Slot    int
}

// Setlist is a named run of slots.
type Setlist struct {
	// Meta is the setlist's metadata, preserved verbatim.
	Meta json.RawMessage
	// Slots holds one preset payload per position.
	Slots []preset.Data
}

// envelope is the outer JSON object, as written on disk.
type envelope struct {
	Schema      string          `json:"schema"`
	Version     int             `json:"version"`
	Meta        json.RawMessage `json:"meta"`
	Encoding    string          `json:"encoding"`
	Compression compression     `json:"compression"`
	EncodedData string          `json:"encoded_data"`
}

// compression describes the payload, and lets a reader detect a truncated or
// corrupted file before trying to parse megabytes of JSON out of it.
type compression struct {
	Type             string `json:"type"`
	CRC32            uint32 `json:"crc32"`
	DecompressedSize int    `json:"decompressed_size"`
}

// payloadSetlist is what a .hls file's encoded_data decodes to.
type payloadSetlist struct {
	Meta    json.RawMessage `json:"meta"`
	Presets []preset.Data   `json:"presets"`
}

// payloadBundle is what a .hlb file's encoded_data decodes to.
type payloadBundle struct {
	Setlists []payloadSetlist `json:"setlists"`
}
