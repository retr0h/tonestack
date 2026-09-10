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

package wire_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

type RPCPublicTestSuite struct {
	suite.Suite
}

// hex decodes a fixture, ignoring the spaces used to group it.
func (s *RPCPublicTestSuite) hex(in string) []byte {
	clean := make([]byte, 0, len(in))

	for i := range len(in) {
		if in[i] != ' ' {
			clean = append(clean, in[i])
		}
	}

	out, err := hex.DecodeString(string(clean))
	s.Require().NoError(err)

	return out
}

// listing is a reply naming two presets.
const listing = "83" + "66cd03e8" + "6700" + "68dc0002" +
	"81cd0000" + "81cd006d" + "ac43542d426c61636b656e6400" +
	"81cd0001" + "81cd006d" + "ab43542d44617920434c4e00"

// TestEncodeRequest writes what a device expects to read.
func (s *RPCPublicTestSuite) TestEncodeRequest() {
	tests := []struct {
		name string
		args []wire.Arg
		// the whole request, byte for byte.
		wantHex string
		// the arguments as a device reads them back.
		wantArgs map[int8]any
		// a document of this many bytes, and the tag it must go out under.
		size int
		tag  byte
	}{
		{
			// Byte-for-byte what HX Edit sends to list presets. The argument
			// order is the order it uses, which is not sorted, and the
			// integers are the narrowest unsigned form that holds them.
			name:    "the request HX Edit sends to list presets",
			args:    []wire.Arg{{Key: 107, Value: 0}, {Key: 101, Value: 2}},
			wantHex: "8366cd03e8640165826b006502",
		},
		{
			// A read sends numbers. A write also sends the preset itself, the
			// name to save it under, and booleans a device echoes back
			// unchanged.
			name: "an argument of every kind",
			args: []wire.Arg{
				wire.Number(107, 0),
				wire.Text(109, "Mike Dirnt"),
				wire.Flag(123, false),
				wire.Blob(110, []byte{0x01, 0x02}),
			},
			wantArgs: map[int8]any{
				// Terminated, because a device reads a name that is not as
				// running on into whatever follows it.
				109: "Mike Dirnt\x00",
				123: false,
				// Under MessagePack's string tag, which is what a device
				// sends a preset document as and what it takes one back as.
				110: "\x01\x02",
			},
		},
		// The tag is the difference between a write that lands and one
		// answered `error -3`. A device sends a preset document tagged str16
		// and takes it back under the same tag; a generic encoder picks the
		// narrowest binary tag that fits, bin16, and the bytes are identical
		// while the tag is not.
		{name: "a short document", size: 10, tag: 0xaa},
		{name: "a preset", size: 2476, tag: 0xda},
		{name: "one past what two bytes count", size: 70000, tag: 0xdb},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			args := tt.args
			if tt.tag != 0 {
				args = []wire.Arg{wire.Blob(110, make([]byte, tt.size))}
			}

			got := wire.EncodeRequest(wire.Request{
				Txn: wire.FirstTxn, Opcode: 8, Args: args,
			})

			if tt.wantHex != "" {
				s.Require().Equal(tt.wantHex, hex.EncodeToString(
					wire.EncodeRequest(wire.Request{
						Txn: wire.FirstTxn, Opcode: 1, Args: args,
					})))

				return
			}

			if tt.tag != 0 {
				// Past the request map, the argument map and the key,
				// whatever widths those took: the document's own tag is the
				// first byte that is not one of them.
				at := bytes.IndexByte(got, tt.tag)
				s.Require().Positive(at, "no %#x tag anywhere in the request", tt.tag)

				s.Require().NotContains(got[:at], byte(0xc5),
					"a binary tag would be the wrong one")

				return
			}

			var doc map[int8]any

			dec := msgpack.NewDecoder(bytes.NewReader(got))
			dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
				return d.DecodeUntypedMap()
			})
			s.Require().NoError(dec.Decode(&doc))

			decoded, ok := doc[101].(map[any]any)
			s.Require().True(ok)

			for key, want := range tt.wantArgs {
				s.Require().Equal(want, decoded[key])
			}
		})
	}
}

// TestDecodeResponse reads what a device answered.
func (s *RPCPublicTestSuite) TestDecodeResponse() {
	tests := []struct {
		name   string
		body   string
		opcode uint64
		// bytes no device would send.
		bad bool

		wantTxn    uint64
		wantStatus wire.Status
		err        error
		errText    []string
	}{
		{
			name:       "a reply carrying a result",
			body:       listing,
			opcode:     1,
			wantTxn:    1000,
			wantStatus: wire.StatusDone,
		},
		{
			// Status 255 with a signed code.
			name:       "a refusal",
			body:       "83" + "66cd03e8" + "67cc ff" + "68" + "81" + "6f" + "d0fd",
			opcode:     6,
			wantTxn:    1000,
			wantStatus: wire.StatusRefused,
			err:        wire.ErrRefused,
			errText:    []string{"opcode 6", "-3"},
		},
		{
			// Status 1 is not a refusal — it means the device took the call
			// and will finish it later. A client reading any non-zero status
			// as failure decides every deferred operation failed.
			name:       "a call the device will finish later",
			body:       "83" + "66cd03e8" + "6701" + "68c0",
			opcode:     20,
			wantTxn:    1000,
			wantStatus: wire.StatusAccepted,
		},
		{
			name:    "nothing at all",
			bad:     true,
			errText: []string{"decoding response"},
		},
		{
			name:    "a value that is not a map",
			body:    "c3",
			bad:     true,
			errText: []string{"expected a map"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := wire.DecodeResponse(s.hex(tt.body))

			if tt.bad {
				s.Require().Error(err)

				for _, want := range tt.errText {
					s.Require().Contains(err.Error(), want)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.wantTxn, got.Txn)
			s.Require().Equal(tt.wantStatus, got.Status)

			err = got.Err(tt.opcode)

			if tt.err == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.err)

			for _, want := range tt.errText {
				s.Require().Contains(err.Error(), want)
			}
		})
	}
}

// TestDecodePresetList reads a listing a device sent.
func (s *RPCPublicTestSuite) TestDecodePresetList() {
	tests := []struct {
		name string
		// a reply to read the listing out of.
		body string
		// or a result handed straight over, for shapes a device would not
		// send.
		result any

		wantNames []string
		errText   string
	}{
		{
			name: "two presets",
			body: listing,
			// The trailing NUL is gone: Line 6's strings are C strings whose
			// declared length counts the terminator.
			wantNames: []string{"CT-Blackend", "CT-Day CLN"},
		},
		{
			// The key an entry carries is the index the preset had before it
			// was last reordered on the pedal. No command accepts it as an
			// address, so a listing has to be read positionally.
			name: "entries keyed by where they used to be",
			body: "83" + "66cd03e8" + "6700" + "68dc0002" +
				"81cd0384" + "81cd006d" + "a64669727374 00" +
				"81cd0001" + "81cd006d" + "a75365636f6e6400",
			wantNames: []string{"First", "Second"},
		},
		{
			name:    "a result that is not an array",
			result:  "nope",
			errText: "expected an array",
		},
		{
			name:    "an entry that is not a map",
			result:  []any{"nope"},
			errText: "expected a map",
		},
		{
			name:    "an entry whose detail is not a map",
			result:  []any{map[any]any{int8(0): "nope"}},
			errText: "expected a detail map",
		},
		{
			name:    "an entry with no name",
			result:  []any{map[any]any{int8(0): map[any]any{int8(123): false}}},
			errText: "no name",
		},
		{
			name:    "a name that is not a string",
			result:  []any{map[any]any{int8(0): map[any]any{int8(109): 42}}},
			errText: "expected a name",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := tt.result

			if tt.body != "" {
				resp, err := wire.DecodeResponse(s.hex(tt.body))
				s.Require().NoError(err)

				result = resp.Result
			}

			got, err := wire.DecodePresetList(result)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got, len(tt.wantNames))

			for i, want := range tt.wantNames {
				s.Require().Equal(want, got[i].Name)
				s.Require().Equal(i, got[i].Slot,
					"an entry is addressed by where it sits")
			}
		})
	}
}

// TestLabel names a slot the way the hardware does.
func (s *RPCPublicTestSuite) TestLabel() {
	tests := []struct {
		slot int
		want string
	}{
		{slot: 0, want: "01A"},
		{slot: 1, want: "01B"},
		{slot: 2, want: "01C"},
		{slot: 3, want: "02A"},
		{slot: 7, want: "03B"},
		{slot: 125, want: "42C"},
	}

	for _, tt := range tests {
		s.Run(tt.want, func() {
			s.Require().Equal(tt.want, wire.Preset{Slot: tt.slot}.Label())
		})
	}
}

func TestRPCPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RPCPublicTestSuite))
}
