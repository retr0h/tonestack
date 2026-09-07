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
	"errors"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/slot"
)

// Map keys in a remote call. Line 6 numbers them rather than naming them.
const (
	keyOpcode = 100
	keyArgs   = 101
	keyTxn    = 102
	keyStatus = 103
	keyResult = 104
	keyError  = 111
)

// FirstTxn is the transaction identifier a channel starts at.
const FirstTxn = 1000

// Status is how a call completed.
type Status uint8

// The completion modes.
//
// Status is not an error code, which is the trap: a client reading any
// non-zero value as failure decides every deferred operation failed.
const (
	// StatusDone means the call finished.
	StatusDone Status = 0
	// StatusAccepted means the device took the call and will finish it
	// later, announcing completion in a notification carrying the same
	// transaction identifier. It is not validation either — asking for
	// preset 999 on a device holding 126 is accepted and does nothing.
	StatusAccepted Status = 1
	// StatusRefused means the device declined.
	StatusRefused Status = 255
)

// Arg is one argument to a call.
//
// Ordered rather than a map, because the device is shown arguments in the
// order HX Edit writes them and this keeps a call byte-identical to a capture
// that is known to work. Sorting them is not the same order.
type Arg struct {
	Key   int
	Value uint64
}

// Request is a call to make.
type Request struct {
	Txn    uint64
	Opcode uint64
	Args   []Arg
}

// Response is what a call returned.
type Response struct {
	Txn    uint64
	Status Status
	Result any
}

// ErrRefused reports a call the device declined.
var ErrRefused = errors.New("device refused the request")

// RefusedError carries the code the device gave.
type RefusedError struct {
	Opcode uint64
	Code   int64
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("device refused the request: opcode %d, error %d",
		e.Opcode, e.Code)
}

func (*RefusedError) Unwrap() error { return ErrRefused }

// EncodeRequest renders a call.
//
// Integers are written wide — 1000 as a three-byte uint16 rather than a
// one-byte fixint — because that is what HX Edit emits, and matching it keeps
// what goes on the wire identical to what the device has been shown to
// accept.
//
// Keys are written in the order HX Edit writes them, for the same reason.
func EncodeRequest(r Request) []byte {
	var buf bytes.Buffer

	// Every write below goes into that buffer, which cannot fail, so none of
	// them is checked. Branches no test can reach would only hide the ones
	// that matter.
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(false)

	_ = enc.EncodeMapLen(3)
	_ = enc.EncodeInt(keyTxn)
	_ = enc.EncodeUint(r.Txn)
	_ = enc.EncodeInt(keyOpcode)
	_ = enc.EncodeUint(r.Opcode)
	_ = enc.EncodeInt(keyArgs)
	_ = enc.EncodeMapLen(len(r.Args))

	for _, a := range r.Args {
		_ = enc.EncodeInt(int64(a.Key))
		// EncodeUint rather than Encode: the generic path writes a value at
		// full width, where the device is sent the narrowest unsigned form
		// that holds it — 1000 as a three-byte uint16, 2 as a single byte.
		_ = enc.EncodeUint(a.Value)
	}

	return buf.Bytes()
}

// DecodeResponse reads a reply.
//
// Only the first value in the body is the reply. Real replies carry an opaque
// tail after it on roughly every other fresh session, and decoding greedily
// turns that into an error where there is none.
func DecodeResponse(body []byte) (Response, error) {
	dec := msgpack.NewDecoder(bytes.NewReader(body))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		return d.DecodeUntypedMap()
	})

	raw, err := dec.DecodeInterface()
	if err != nil {
		return Response{}, fmt.Errorf("decoding response: %w", err)
	}

	m, ok := raw.(map[any]any)
	if !ok {
		return Response{}, fmt.Errorf("decoding response: expected a map, got %T", raw)
	}

	txn, _ := asUint(m[int8(keyTxn)])
	status, _ := asUint(m[int8(keyStatus)])

	return Response{
		Txn:    txn,
		Status: Status(status),
		Result: m[int8(keyResult)],
	}, nil
}

// Err returns the failure a response carries, if it is one.
func (r Response) Err(opcode uint64) error {
	if r.Status != StatusRefused {
		return nil
	}

	code := int64(0)
	if m, ok := r.Result.(map[any]any); ok {
		if v, ok := asInt(m[int8(keyError)]); ok {
			code = v
		}
	}

	return &RefusedError{Opcode: opcode, Code: code}
}

// Preset is one slot as the device reports it.
type Preset struct {
	// Slot is the position in the setlist, counted from zero.
	Slot int
	// Name is what the device shows, with the terminator removed.
	Name string
}

// Label renders a slot the way the hardware labels it — 01A through 42C.
//
// A player reading this is looking at the pedal, where a bare index would
// mean counting.
func (p Preset) Label() string {
	return slot.Label(p.Slot)
}

// keyPresetName is where a name sits inside a listing entry.
const keyPresetName = 109

// DecodePresetList reads the reply to a list-presets call.
//
// Entries are read by position, not by the key each carries. That key is the
// index a preset had before it was last reordered on the pedal, and no
// command accepts it as an address — passing one through as a slot number is
// how a device gets sent somewhere that does not exist.
func DecodePresetList(result any) ([]Preset, error) {
	rows, ok := result.([]any)
	if !ok {
		return nil, fmt.Errorf("listing presets: expected an array, got %T", result)
	}

	out := make([]Preset, 0, len(rows))

	for i, row := range rows {
		name, err := presetName(row)
		if err != nil {
			return nil, fmt.Errorf("listing presets: slot %d: %w", i, err)
		}

		out = append(out, Preset{Slot: i, Name: name})
	}

	return out, nil
}

// presetName digs the name out of one listing entry.
//
// An entry is a map of exactly one pair, so the detail sits one level in
// whatever the key turns out to be.
func presetName(row any) (string, error) {
	entry, ok := row.(map[any]any)
	if !ok {
		return "", fmt.Errorf("expected a map, got %T", row)
	}

	for _, detail := range entry {
		fields, ok := detail.(map[any]any)
		if !ok {
			return "", fmt.Errorf("expected a detail map, got %T", detail)
		}

		for k, v := range fields {
			if n, ok := asUint(k); !ok || n != keyPresetName {
				continue
			}

			name, ok := asString(v)
			if !ok {
				return "", fmt.Errorf("expected a name, got %T", v)
			}

			return name, nil
		}
	}

	return "", errors.New("no name in this entry")
}
