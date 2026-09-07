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

package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// ErrNotADocument reports bytes that are not a preset as a device stores one.
var ErrNotADocument = errors.New("not a preset document")

// offsetCount is how many byte offsets a preset carries.
const offsetCount = 12

// offsetOrder is what each of those offsets points at.
//
// The first is the preset map itself; the last two are the end of the
// document. Everything between points at the key byte of one section, in this
// order, which is not the order the sections are written in. Read off three
// presets from an HX Stomp, identical in all three.
var offsetOrder = []int8{0, 1, 3, 4, 2, 5, 6, 7, 10}

// Document is a preset as a device stores it, kept in the form it arrived.
//
// Every section is held as the bytes the device sent. That is the whole point:
// MessagePack can write the same number several widths, the device writes wide
// where an encoder writes narrow, and it seeks by a table of byte offsets
// rather than walking the document. Re-encoding a preset changes its length by
// about a hundred bytes, every offset after the change points at the wrong
// thing, and the device accepts the write and then reads the preset as empty.
//
// Keeping the bytes means a preset survives being read and written, and a
// change to one section moves only what follows it.
type Document struct {
	// header is the magic string and the offset table, exactly as they
	// arrived. The table is rewritten on encoding; the bytes around it are
	// not.
	magic []byte
	table []byte
	// order is the sections in the order the device wrote them, which is not
	// the order the offset table lists them in.
	order []int8
	// sections is each one's body, untouched.
	sections map[int8]msgpack.RawMessage
}

// DecodeDocument reads a preset without interpreting it.
func DecodeDocument(raw []byte) (*Document, error) {
	dec := msgpack.NewDecoder(bytes.NewReader(raw))

	magic, err := decodeRawString(dec)
	if err != nil || string(magic) != magicHeader {
		return nil, fmt.Errorf("%w: no %q header", ErrNotADocument, magicHeader)
	}

	table, err := decodeRawString(dec)
	if err != nil || len(table) != offsetCount*4 {
		return nil, fmt.Errorf("%w: no offset table", ErrNotADocument)
	}

	n, err := dec.DecodeMapLen()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotADocument, err)
	}

	out := &Document{
		magic:    magic,
		table:    table,
		order:    make([]int8, 0, n),
		sections: make(map[int8]msgpack.RawMessage, n),
	}

	for range n {
		key, err := dec.DecodeInt8()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNotADocument, err)
		}

		body, err := dec.DecodeRaw()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNotADocument, err)
		}

		out.order = append(out.order, key)
		out.sections[key] = body
	}

	return out, nil
}

// Section returns one section's bytes, or nothing when the preset has none.
func (d *Document) Section(key int8) (msgpack.RawMessage, bool) {
	body, ok := d.sections[key]

	return body, ok
}

// SetSection replaces one section, leaving every other byte alone.
//
// A section the preset did not have is added at the end, which is where a
// device puts one it did not have either.
func (d *Document) SetSection(key int8, body msgpack.RawMessage) {
	if _, ok := d.sections[key]; !ok {
		d.order = append(d.order, key)
	}

	d.sections[key] = body
}

// Encode writes the preset back, with the offset table recomputed.
//
// A document nobody changed encodes to the bytes it was read from.
func (d *Document) Encode() []byte {
	var body bytes.Buffer

	// Written by hand rather than through an encoder: the map header and the
	// keys have to come out the width the device wrote them, and an encoder
	// picks its own.
	body.Write(mapHeader(len(d.order)))

	at := make(map[int8]int, len(d.order))

	for _, key := range d.order {
		at[key] = body.Len()
		body.WriteByte(byte(key))
		body.Write(d.sections[key])
	}

	// The header is a fixed length: a nine-byte magic and a table the device
	// always writes as a str16. So where the document starts is known before
	// the table that describes it is written.
	start := len(header(d.magic, d.table))

	out := header(d.magic, d.offsets(at, start, body.Len()))

	return append(out, body.Bytes()...)
}

// offsets builds the table pointing at where each section landed.
func (d *Document) offsets(at map[int8]int, start, size int) []byte {
	table := make([]byte, len(d.table))

	// The first offset is the map itself, and the last two are the end.
	put(table, 0, start)
	put(table, offsetCount-2, start+size)
	put(table, offsetCount-1, start+size)

	for i, key := range offsetOrder {
		where, ok := at[key]
		if !ok {
			continue
		}

		put(table, i+1, start+where)
	}

	return table
}

// put writes one offset.
func put(table []byte, i, v int) {
	binary.LittleEndian.PutUint32(table[i*4:], uint32(v))
}

// header renders the two values a preset opens with.
func header(magic, table []byte) []byte {
	var out bytes.Buffer

	out.Write(strHeader(len(magic)))
	out.Write(magic)
	out.Write(strHeader(len(table)))
	out.Write(table)

	return out.Bytes()
}

// magicHeader is what a preset from a device starts with.
const magicHeader = "l6-helix\x00"

// decodeRawString reads a string as the bytes it holds.
//
// A preset's magic and offset table are carried in MessagePack's string type
// and neither is text. Decoding them as strings would mangle every byte above
// 0x7f.
func decodeRawString(dec *msgpack.Decoder) ([]byte, error) {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return nil, err
	}

	// Past the length prefix, whichever width it was written in.
	switch {
	case len(raw) > 0 && raw[0]&0xe0 == 0xa0:
		return raw[1:], nil
	case len(raw) > 1 && raw[0] == 0xd9:
		return raw[2:], nil
	case len(raw) > 2 && raw[0] == 0xda:
		return raw[3:], nil
	default:
		return nil, fmt.Errorf("%w: not a string", ErrNotADocument)
	}
}

// strHeader renders a string's length the way the device writes it.
//
// Only two strings go through here: the nine-byte magic, which is a fixstr,
// and the 48-byte offset table, which the device writes as a str16 where an
// encoder would choose str8. One byte of difference there moves every offset.
func strHeader(n int) []byte {
	if n < 32 {
		return []byte{byte(0xa0 | n)}
	}

	return []byte{0xda, byte(n >> 8), byte(n)}
}

// mapHeader renders a map's length the way the device writes it.
func mapHeader(n int) []byte {
	if n < 16 {
		return []byte{byte(0x80 | n)}
	}

	return []byte{0xde, byte(n >> 8), byte(n)}
}
