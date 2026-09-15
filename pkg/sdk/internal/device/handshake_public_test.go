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

package device_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// HandshakePublicTestSuite covers making a request and matching the answer to it.
type HandshakePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *HandshakePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// answer encodes what a device replies to one transaction.
func (s *HandshakePublicTestSuite) answer(
	txn uint64,
	status int,
	result any,
) []byte {
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
func (s *HandshakePublicTestSuite) reply(
	txn uint64,
	status int,
	result any,
) []byte {
	return device.Reply(device.ControlChannel, s.answer(txn, status, result))
}

// replyOnData is the answer to a question about a preset document, which the
// device takes on the data channel rather than the control one.
func (s *HandshakePublicTestSuite) replyOnData(
	txn uint64,
	status int,
	result any,
) []byte {
	return device.Reply(device.DataChannel, s.answer(txn, status, result))
}

// session returns one with its channels already open, over a scripted device.
func (s *HandshakePublicTestSuite) session(
	d *deviceDouble,
) *device.Session {
	out := device.NewTestSession(s.T(), d.out, d.in)
	out.OpenChannels()

	return out
}

func (s *HandshakePublicTestSuite) TestCall() {
	broken := errors.New("the bus went away")

	// One answer too long for a transfer, split in three the way a device
	// sends it.
	long := string(bytes.Repeat([]byte("a long answer "), 50))
	envelope := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromDevice, Service: 2, Body: s.answer(device.FirstTxn, 0, long),
	})
	third := len(envelope) / 3
	parts := [][]byte{envelope[:third], envelope[third : 2*third], envelope[2*third:]}

	tests := []struct {
		name      string
		channel   string
		device    func() (*deviceDouble, device.TestSender)
		cancelled bool
		want      any
		err       error
		message   string
		// how many reads the call took, when that is the point.
		reads int
		// acks is what every acknowledgement the call sent on its channel
		// carried, in order, when that is the point.
		acks []uint32
	}{
		{
			// A bus that has gone is not a device with nothing to say. Read
			// as silence, it waited out the whole reply budget and then said
			// no reply, which sends somebody looking at the wrong thing.
			name:    "a device whose read fails outright",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := readFails(s.ctrl, broken)

				return d, d.out
			},
			err:     broken,
			message: "reading from the device",
			reads:   1,
		},
		{
			// Cancellation is not silence. Reporting it as "no reply" told
			// somebody who pressed Ctrl-C that their device had stopped
			// answering, six seconds after they stopped waiting.
			name:    "a call nobody is left waiting for",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl)

				return d, d.out
			},
			cancelled: true,
			err:       context.Canceled,
			message:   "context canceled",
		},
		{
			// A length no frame carries. The buffer is out of step with the
			// stream and no later byte brings it back, so a channel that
			// held it answered nothing again for the rest of the session.
			// The answer behind it is read once the unreadable bytes are
			// dropped.
			name:    "a frame claiming a length nothing could hold",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl,
					device.FrameFor(device.ControlChannel, wire.MsgData,
						[]byte{1, 0, 5, 0, 0xff, 0xff, 0xff, 0xff}),
					s.reply(device.FirstTxn, 0, "an answer"),
				)

				return d, d.out
			},
			want: "an answer",
		},
		{
			name:    "a device that says nothing at all",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl)

				return d, d.out
			},
			message: "no reply to opcode 1",
		},
		{
			name:    "an answer to this call",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl, s.reply(device.FirstTxn, 0, "done"))

				return d, d.out
			},
			want: "done",
		},
		{
			// A long answer arrives a transfer at a time, and the device sends
			// the next only once the host has acknowledged the last: the
			// double releases one part per frame the session writes. The
			// last part completes the answer, so nothing acknowledges it
			// inside the call.
			name:    "an answer split across three transfers",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl,
					device.FrameFor(device.ControlChannel, wire.MsgData, parts[0]),
					device.FrameFor(device.ControlChannel, wire.MsgData, parts[1]),
					device.FrameFor(device.ControlChannel, wire.MsgData, parts[2]),
				)

				return d, d.out
			},
			want: long,
			acks: []uint32{
				wire.AckBase + uint32(len(parts[0])),
				wire.AckBase + uint32(len(parts[0])+len(parts[1])),
			},
		},
		{
			// A notification carries no transaction and is not anybody's
			// reply. Letting one be mistaken for this reply would answer the
			// wrong question.
			name:    "somebody else's answer, skipped",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl,
					s.reply(device.FirstTxn+99, 0, "not yours"),
					s.reply(device.FirstTxn, 0, "yours"),
				)

				return d, d.out
			},
			want: "yours",
		},
		{
			// Whatever arrives is not guaranteed to be a reply.
			name:    "an answer that will not decode, skipped",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl,
					device.Reply(device.ControlChannel, []byte{0xc1}),
					s.reply(device.FirstTxn, 0, "yours"),
				)

				return d, d.out
			},
			want: "yours",
		},
		{
			name:    "a channel nobody opened",
			channel: "nowhere",
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl)

				return d, d.out
			},
			message: "no nowhere channel",
		},
		{
			name:    "a bus that will not take the request",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl)

				return d, writeFails(s.ctrl, errors.New("boom")).out
			},
			message: "boom",
		},
		{
			// Bytes that arrived are acknowledged, and a device that stops
			// listening at that point has to be reported: an unacknowledged
			// stream stalls.
			name:    "a bus that will not take the acknowledgement",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl, device.Reply(device.ControlChannel, []byte{0xc1}))

				return d, &device.FailAfter{Sender: d.out, OK: 1, Err: errors.New("boom")}
			},
			message: "boom",
		},
		{
			// Status 255 is a refusal, and the code it carries is the useful
			// half.
			name:    "a device that refuses",
			channel: device.ControlChannel,
			device: func() (*deviceDouble, device.TestSender) {
				d := answers(s.ctrl, s.reply(device.FirstTxn, 255, map[int]int{111: 7}))

				return d, d.out
			},
			err:     wire.ErrRefused,
			message: "opcode 1",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			in, out := tc.device()

			session := device.NewTestSession(s.T(), out, in.in)
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
				s.Require().Equal(uint64(device.FirstTxn), got.Txn)

				if tc.acks != nil {
					s.Require().Equal(tc.acks, ackValues(in, tc.channel),
						"each part acknowledged before the next, and the last not at all")
				}

				return
			}

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)

			if tc.err != nil {
				s.Require().ErrorIs(err, tc.err)
			}

			if tc.reads > 0 {
				s.Require().Equal(tc.reads, in.readCount(), "within one read")
				s.Require().NotContains(err.Error(), "no reply")
			}
		})
	}
}

// TestHandshake opens every channel, once, and stops at the first frame the
// bus will not take. A handshake is never retried, so a failure partway
// through is reported rather than papered over.
func (s *HandshakePublicTestSuite) TestHandshake() {
	tests := []struct {
		name string
		// how many frames the bus takes before it refuses the rest. Negative
		// means it takes them all.
		sendsOK   int
		cancelled bool
		says      string
	}{
		{name: "a device that takes every frame", sendsOK: -1},
		{
			// Somebody who stopped waiting is told so at the first pause,
			// rather than the channels being opened for nobody.
			name:      "a caller who stopped waiting",
			sendsOK:   -1,
			cancelled: true,
			says:      "context canceled",
		},
		{
			// The hello goes out, and the frame that names the service does
			// not.
			name:    "a bus that refuses the opening of a service",
			sendsOK: 1,
			says:    "boom",
		},
		{
			// The control channel serves two services, closed and reopened
			// between them. A close that does not go out leaves the device
			// talking about the first.
			name:    "a bus that refuses the close between two services",
			sendsOK: 3,
			says:    "boom",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := answers(s.ctrl)

			out := device.TestSender(d.out)
			if tt.sendsOK >= 0 {
				out = &device.FailAfter{Sender: d.out, OK: tt.sendsOK, Err: errors.New("boom")}
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			err := device.NewTestSession(s.T(), out, d.in).Handshake(ctx)

			if tt.says == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

func (s *HandshakePublicTestSuite) TestPresets() {
	tests := []struct {
		name   string
		device func() *deviceDouble
		want   []wire.Preset
		fails  bool
	}{
		{
			// Read by position, not by the key each entry carries: that key
			// is the index a preset had before it was last reordered on the
			// pedal.
			name: "what a setlist holds",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.reply(device.FirstTxn, 0, []any{
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
			device: func() *deviceDouble { return writeFails(s.ctrl, errors.New("boom")) },
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

// TestReadPreset covers what a slot answers with.
//
// The three answers are a document, nothing at all, and something else. The
// last one used to reach the caller as an `any` nobody had checked, where a
// failed type assertion read as an empty slot.
func (s *HandshakePublicTestSuite) TestReadPreset() {
	tests := []struct {
		name   string
		device func() *deviceDouble
		want   []byte
		// the shape the failure must name.
		shape string
		fails bool
	}{
		{
			name: "a slot holding a preset",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.replyOnData(device.FirstTxn, 0, "a preset"))
			},
			want: []byte("a preset"),
		},
		{
			// A slot holding nothing is not a failure, and a backup has to
			// know the difference.
			name: "a slot holding nothing",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.replyOnData(device.FirstTxn, 0, nil))
			},
		},
		{
			name: "an answer that is not a preset",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.replyOnData(device.FirstTxn, 0, map[int]int{1: 2}))
			},
			shape: "map with 1 keys",
		},
		{
			name:   "a bus that will not answer",
			device: func() *deviceDouble { return writeFails(s.ctrl, errors.New("boom")) },
			fails:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.session(tt.device()).ReadPreset(context.Background(), 0, 3)

			if tt.fails {
				s.Require().Error(err)

				return
			}

			if tt.shape != "" {
				s.Require().ErrorIs(err, device.ErrNotAPreset)

				var answer *device.NotAPresetError
				s.Require().ErrorAs(err, &answer)
				s.Require().Equal(tt.shape, answer.Shape())

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

func TestHandshakeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandshakePublicTestSuite))
}
