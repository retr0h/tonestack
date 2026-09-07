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

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

type RPCPublicTestSuite struct {
	suite.Suite
}

func (s *RPCPublicTestSuite) TestEncodesTheCapturedRequest() {
	// Byte-for-byte what HX Edit sends to list presets. The argument order is
	// the order it uses, which is not sorted, and the integers are the
	// narrowest unsigned form that holds them.
	got := wire.EncodeRequest(wire.Request{
		Txn: wire.FirstTxn, Opcode: 1,
		Args: []wire.Arg{{Key: 107, Value: 0}, {Key: 101, Value: 2}},
	})

	s.Require().Equal("8366cd03e8640165826b006502", hex.EncodeToString(got))
}

func (s *RPCPublicTestSuite) TestDecodesAReply() {
	body := s.hex("83" + "66cd03e8" + "6700" + "68dc0002" +
		"81cd0000" + "81cd006d" + "ac43542d426c61636b656e6400" +
		"81cd0001" + "81cd006d" + "ab43542d44617920434c4e00")

	got, err := wire.DecodeResponse(body)

	s.Require().NoError(err)
	s.Require().Equal(uint64(1000), got.Txn)
	s.Require().Equal(wire.StatusDone, got.Status)
	s.Require().NoError(got.Err(1))
}

func (s *RPCPublicTestSuite) TestDecodesThePresetList() {
	body := s.hex("83" + "66cd03e8" + "6700" + "68dc0002" +
		"81cd0000" + "81cd006d" + "ac43542d426c61636b656e6400" +
		"81cd0001" + "81cd006d" + "ab43542d44617920434c4e00")

	resp, err := wire.DecodeResponse(body)
	s.Require().NoError(err)

	got, err := wire.DecodePresetList(resp.Result)

	s.Require().NoError(err)
	s.Require().Len(got, 2)
	// The trailing NUL is gone: Line 6's strings are C strings whose declared
	// length counts the terminator.
	s.Require().Equal("CT-Blackend", got[0].Name)
	s.Require().Equal("CT-Day CLN", got[1].Name)
	s.Require().Equal(0, got[0].Slot)
	s.Require().Equal(1, got[1].Slot)
}

func (s *RPCPublicTestSuite) TestReadsEntriesByPositionNotByKey() {
	// The key an entry carries is the index the preset had before it was last
	// reordered on the pedal. No command accepts it as an address, so a
	// listing has to be read positionally.
	body := s.hex("83" + "66cd03e8" + "6700" + "68dc0002" +
		"81cd0384" + "81cd006d" + "a64669727374 00" +
		"81cd0001" + "81cd006d" + "a75365636f6e6400")

	resp, err := wire.DecodeResponse(body)
	s.Require().NoError(err)

	got, err := wire.DecodePresetList(resp.Result)

	s.Require().NoError(err)
	s.Require().Equal(0, got[0].Slot, "the first entry is slot zero whatever it is keyed by")
	s.Require().Equal("First", got[0].Name)
}

func (s *RPCPublicTestSuite) TestLabelsSlotsTheWayTheHardwareDoes() {
	tests := []struct {
		slot int
		want string
	}{
		{0, "01A"},
		{1, "01B"},
		{2, "01C"},
		{3, "02A"},
		{7, "03B"},
		{125, "42C"},
	}

	for _, tc := range tests {
		s.Run(tc.want, func() {
			s.Require().Equal(tc.want, wire.Preset{Slot: tc.slot}.Label())
		})
	}
}

func (s *RPCPublicTestSuite) TestReportsARefusal() {
	// Status 255 with a signed code. Status 1 is not a refusal — it means the
	// device took the call and will finish it later.
	body := s.hex("83" + "66cd03e8" + "67cc ff" + "68" + "81" + "6f" + "d0fd")

	got, err := wire.DecodeResponse(body)
	s.Require().NoError(err)
	s.Require().Equal(wire.StatusRefused, got.Status)

	err = got.Err(6)
	s.Require().ErrorIs(err, wire.ErrRefused)
	s.Require().Contains(err.Error(), "opcode 6")
	s.Require().Contains(err.Error(), "-3")
}

func (s *RPCPublicTestSuite) TestAcceptedIsNotAFailure() {
	body := s.hex("83" + "66cd03e8" + "6701" + "68c0")

	got, err := wire.DecodeResponse(body)

	s.Require().NoError(err)
	s.Require().Equal(wire.StatusAccepted, got.Status)
	s.Require().NoError(got.Err(20),
		"a client reading any non-zero status as failure decides every "+
			"deferred operation failed")
}

func (s *RPCPublicTestSuite) TestRefusesWhatIsNotAReply() {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"nothing at all", "", "decoding response"},
		{"a value that is not a map", "c3", "expected a map"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := wire.DecodeResponse(s.hex(tc.body))

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.want)
		})
	}
}

func (s *RPCPublicTestSuite) TestRefusesAListingItCannotRead() {
	tests := []struct {
		name   string
		result any
		want   string
	}{
		{"a result that is not an array", "nope", "expected an array"},
		{"an entry that is not a map", []any{"nope"}, "expected a map"},
		{
			"an entry whose detail is not a map",
			[]any{map[any]any{int8(0): "nope"}},
			"expected a detail map",
		},
		{
			"an entry with no name",
			[]any{map[any]any{int8(0): map[any]any{int8(123): false}}},
			"no name",
		},
		{
			"a name that is not a string",
			[]any{map[any]any{int8(0): map[any]any{int8(109): 42}}},
			"expected a name",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := wire.DecodePresetList(tc.result)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.want)
		})
	}
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

func (s *RPCPublicTestSuite) TestAnArgumentOfEveryKind() {
	// A read sends numbers. A write also sends the preset itself, the name to
	// save it under, and three booleans a device echoes back unchanged.
	got := wire.EncodeRequest(wire.Request{
		Txn: 1000, Opcode: 8,
		Args: []wire.Arg{
			wire.Number(107, 0),
			wire.Text(109, "Mike Dirnt"),
			wire.Flag(123, false),
			wire.Blob(110, []byte{0x01, 0x02}),
		},
	})

	var doc map[int8]any

	dec := msgpack.NewDecoder(bytes.NewReader(got))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		return d.DecodeUntypedMap()
	})
	s.Require().NoError(dec.Decode(&doc))

	args, ok := doc[101].(map[any]any)
	s.Require().True(ok)

	// Terminated, because a device reads a name that is not as running on
	// into whatever follows it.
	s.Require().Equal("Mike Dirnt\x00", args[int8(109)])
	s.Require().Equal(false, args[int8(123)])
	s.Require().Equal([]byte{0x01, 0x02}, args[int8(110)])
}

func TestRPCPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RPCPublicTestSuite))
}
