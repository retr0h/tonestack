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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// SelectPublicTestSuite covers loading a preset.
//
// The wait is what this file is about. A select is deferred, and a caller
// that returns before the device has finished leaves it holding a half
// finished switch, which it settles by wiping its edit buffer: the preset
// comes up with no blocks and no footswitch colours. Asking the device what
// is loaded, until it says the right thing, is the only honest signal.
type SelectPublicTestSuite struct {
	suite.Suite
}

func (s *SelectPublicTestSuite) SetupTest() {
	poll, budget := *device.SelectPoll, *device.SelectBudget
	*device.SelectPoll = time.Millisecond
	*device.SelectBudget = 50 * time.Millisecond

	s.T().Cleanup(func() {
		*device.SelectPoll, *device.SelectBudget = poll, budget
	})
}

// status is the device answering with one status and nothing else.
func (s *SelectPublicTestSuite) status(txn uint64, status int) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(int64(status)))

	return device.Reply(device.DataChannel, buf.Bytes())
}

// document is the device answering with a preset.
func (s *SelectPublicTestSuite) document(txn uint64, body string) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.EncodeString(body))

	return device.Reply(device.DataChannel, buf.Bytes())
}

// took is the device saying it has taken a request.
func (s *SelectPublicTestSuite) took(txn uint64) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(1))

	return device.Reply(device.DataChannel, buf.Bytes())
}

// playing is the device saying which preset it has loaded.
func (s *SelectPublicTestSuite) playing(txn uint64, setlist, slot int) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(107))
	s.Require().NoError(enc.EncodeInt(int64(setlist)))
	s.Require().NoError(enc.EncodeInt(108))
	s.Require().NoError(enc.EncodeInt(int64(slot)))
	s.Require().NoError(enc.EncodeInt(109))
	s.Require().NoError(enc.EncodeString("Chunky Monkey"))

	return device.Reply(device.DataChannel, buf.Bytes())
}

func (s *SelectPublicTestSuite) session(d *scripted) *device.Session {
	out := device.NewTestSession(d, d)
	out.OpenChannels()

	return out
}

// TestSelectPreset loads a preset and waits for the device to say it landed.
func (s *SelectPublicTestSuite) TestSelectPreset() {
	tests := []struct {
		name      string
		device    func() *scripted
		cancelled bool
		// a caller who stops waiting after the device has answered, rather
		// than before it was asked.
		timeout time.Duration
		says    string
	}{
		{
			// The device reports the preset it was playing before answering
			// with the one that was asked for, which is the window a caller
			// must not return inside.
			name: "one that answers with the old preset first",
			device: func() *scripted {
				return answers(
					s.took(device.FirstTxn),
					s.playing(device.FirstTxn+1, 0, 5),
					s.playing(device.FirstTxn+2, 0, 99),
				)
			},
		},
		{
			name: "one that takes it and never gets there",
			device: func() *scripted {
				return answers(s.took(device.FirstTxn), s.playing(device.FirstTxn+1, 0, 5))
			},
			says: "did not finish switching",
		},
		{
			name:   "one that never takes it at all",
			device: func() *scripted { return answers() },
			says:   "no reply",
		},
		{
			name: "an answer the protocol does not describe",
			device: func() *scripted {
				return answers(s.status(device.FirstTxn, 7))
			},
			says: "unexpected status 7",
		},
		{
			name: "a caller that gave up waiting",
			device: func() *scripted {
				return answers(s.took(device.FirstTxn), s.playing(device.FirstTxn+1, 0, 5))
			},
			cancelled: true,
			says:      "context canceled",
		},
		{
			// And one who gave up while it was still switching: the device
			// took the request and is playing something else, so the wait
			// ends where somebody stopped waiting rather than at the budget.
			name: "a caller that gave up part way through",
			device: func() *scripted {
				return answers(s.took(device.FirstTxn), s.playing(device.FirstTxn+1, 0, 5))
			},
			timeout: 5 * time.Millisecond,
			says:    "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			if tt.timeout > 0 {
				var stop context.CancelFunc

				ctx, stop = context.WithTimeout(context.Background(), tt.timeout)
				defer stop()
			}

			d := tt.device()
			err := s.session(d).SelectPreset(ctx, 0, 99)

			if tt.says == "" {
				s.Require().NoError(err)
				s.Require().Empty(d.replies, "every answer was read")

				return
			}

			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// TestLoaded reads which preset the device is playing, which is the only
// honest signal that a switch has finished.
func (s *SelectPublicTestSuite) TestLoaded() {
	tests := []struct {
		name   string
		device func() *scripted
		want   wire.Loaded
		err    bool
	}{
		{
			name: "a device that says",
			device: func() *scripted {
				return answers(s.playing(device.FirstTxn, 0, 99))
			},
			want: wire.Loaded{Setlist: 0, Slot: 99, Name: "Chunky Monkey"},
		},
		{
			name:   "one that will not",
			device: func() *scripted { return answers() },
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.session(tt.device()).Loaded(context.Background())

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestReadCurrent reads the document a device is playing, which is the edit
// buffer rather than a stored slot.
func (s *SelectPublicTestSuite) TestReadCurrent() {
	tests := []struct {
		name   string
		device func() *scripted
		want   []byte
		err    bool
	}{
		{
			name: "a device holding a preset",
			device: func() *scripted {
				return answers(s.document(device.FirstTxn, "a preset"))
			},
			want: []byte("a preset"),
		},
		{
			name:   "one that will not answer",
			device: func() *scripted { return answers() },
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.session(tt.device()).ReadCurrent(context.Background())

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

func TestSelectTestSuite(t *testing.T) {
	suite.Run(t, new(SelectPublicTestSuite))
}
