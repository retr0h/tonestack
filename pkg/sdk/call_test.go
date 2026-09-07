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

package sdk_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// CallTestSuite covers making a request and matching the answer to it.
type CallTestSuite struct {
	suite.Suite
}

// answer encodes what a device replies to one transaction.
func (s *CallTestSuite) answer(txn uint64, status int, result any) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(int64(status)))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.Encode(result))

	return buf.Bytes()
}

// reply frames an answer the way the device sends it.
func (s *CallTestSuite) reply(txn uint64, status int, result any) []byte {
	return sdk.Reply(sdk.ControlChannel, s.answer(txn, status, result))
}

// replyOnData is the answer to a question about a preset document, which the
// device takes on the data channel rather than the control one.
func (s *CallTestSuite) replyOnData(txn uint64, status int, result any) []byte {
	return sdk.Reply(sdk.DataChannel, s.answer(txn, status, result))
}

// session returns one with its channels already open, over a scripted device.
func (s *CallTestSuite) session(d *device) *sdk.Session {
	out := sdk.NewTestSession(d, d)
	out.OpenChannels()

	return out
}

func (s *CallTestSuite) TestCall() {
	tests := []struct {
		name      string
		channel   string
		device    func() (*device, sdk.TestSender)
		cancelled bool
		want      any
		err       error
		message   string
	}{
		{
			name:      "a call nobody is left waiting for",
			channel:   sdk.ControlChannel,
			device:    func() (*device, sdk.TestSender) { d := answers(); return d, d },
			cancelled: true,
			message:   "opcode 1",
		},
		{
			name:    "an answer to this call",
			channel: sdk.ControlChannel,
			device: func() (*device, sdk.TestSender) {
				d := answers(s.reply(sdk.FirstTxn, 0, "done"))

				return d, d
			},
			want: "done",
		},
		{
			// A notification carries no transaction and is not anybody's
			// reply. Letting one be mistaken for this reply would answer the
			// wrong question.
			name:    "somebody else's answer, skipped",
			channel: sdk.ControlChannel,
			device: func() (*device, sdk.TestSender) {
				d := answers(
					s.reply(sdk.FirstTxn+99, 0, "not yours"),
					s.reply(sdk.FirstTxn, 0, "yours"),
				)

				return d, d
			},
			want: "yours",
		},
		{
			// Whatever arrives is not guaranteed to be a reply.
			name:    "an answer that will not decode, skipped",
			channel: sdk.ControlChannel,
			device: func() (*device, sdk.TestSender) {
				d := answers(
					sdk.Reply(sdk.ControlChannel, []byte{0xc1}),
					s.reply(sdk.FirstTxn, 0, "yours"),
				)

				return d, d
			},
			want: "yours",
		},
		{
			name:    "a channel nobody opened",
			channel: "nowhere",
			device:  func() (*device, sdk.TestSender) { d := answers(); return d, d },
			message: "no nowhere channel",
		},
		{
			name:    "a bus that will not take the request",
			channel: sdk.ControlChannel,
			device: func() (*device, sdk.TestSender) {
				d := answers()

				return d, &device{writeErr: errors.New("boom")}
			},
			message: "boom",
		},
		{
			// Bytes that arrived are acknowledged, and a device that stops
			// listening at that point has to be reported: an unacknowledged
			// stream stalls.
			name:    "a bus that will not take the acknowledgement",
			channel: sdk.ControlChannel,
			device: func() (*device, sdk.TestSender) {
				d := answers(sdk.Reply(sdk.ControlChannel, []byte{0xc1}))

				return d, &sdk.FailAfter{Sender: d, OK: 1, Err: errors.New("boom")}
			},
			message: "boom",
		},
		{
			// Status 255 is a refusal, and the code it carries is the useful
			// half.
			name:    "a device that refuses",
			channel: sdk.ControlChannel,
			device: func() (*device, sdk.TestSender) {
				d := answers(s.reply(sdk.FirstTxn, 255, map[int]int{111: 7}))

				return d, d
			},
			err:     wire.ErrRefused,
			message: "opcode 1",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			in, out := tc.device()

			session := sdk.NewTestSession(out, in)
			session.OpenChannels()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tc.cancelled {
				cancel()
			}

			got, err := session.Call(ctx, tc.channel, 1, nil)

			if tc.message == "" {
				s.Require().NoError(err)
				s.Require().Equal(tc.want, got.Result)
				s.Require().Equal(uint64(sdk.FirstTxn), got.Txn)

				return
			}

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)

			if tc.err != nil {
				s.Require().ErrorIs(err, tc.err)
			}
		})
	}
}

func (s *CallTestSuite) TestPresets() {
	tests := []struct {
		name   string
		device func() *device
		want   []wire.Preset
		fails  bool
	}{
		{
			// Read by position, not by the key each entry carries: that key
			// is the index a preset had before it was last reordered on the
			// pedal.
			name: "what a setlist holds",
			device: func() *device {
				return answers(s.reply(sdk.FirstTxn, 0, []any{
					map[int]any{0: map[int]any{109: "Chunky Monkey\x00"}},
					map[int]any{1: map[int]any{109: "Fat Mike\x00"}},
				}))
			},
			want: []wire.Preset{
				{Slot: 0, Name: "Chunky Monkey"},
				{Slot: 1, Name: "Fat Mike"},
			},
		},
		{
			name:   "a bus that will not answer",
			device: func() *device { return &device{writeErr: errors.New("boom")} },
			fails:  true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, err := s.session(tc.device()).Presets(context.Background(), 0)

			if tc.fails {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tc.want, got)
		})
	}
}

func (s *CallTestSuite) TestReadPreset() {
	tests := []struct {
		name   string
		device func() *device
		want   any
		fails  bool
	}{
		{
			name: "one slot",
			device: func() *device {
				return answers(s.replyOnData(sdk.FirstTxn, 0, "a preset"))
			},
			want: "a preset",
		},
		{
			name:   "a bus that will not answer",
			device: func() *device { return &device{writeErr: errors.New("boom")} },
			fails:  true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, err := s.session(tc.device()).ReadPreset(context.Background(), 0, 3)

			if tc.fails {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tc.want, got)
		})
	}
}

func TestCallTestSuite(t *testing.T) {
	suite.Run(t, new(CallTestSuite))
}
