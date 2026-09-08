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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// SelectTestSuite covers loading a preset.
//
// The wait is what this file is about. A select is deferred, and a caller
// that returns before the device has finished leaves it holding a half
// finished switch, which it settles by wiping its edit buffer: the preset
// comes up with no blocks and no footswitch colours. Asking the device what
// is loaded, until it says the right thing, is the only honest signal.
type SelectTestSuite struct {
	suite.Suite
}

func (s *SelectTestSuite) SetupTest() {
	poll, budget := *sdk.SelectPoll, *sdk.SelectBudget
	*sdk.SelectPoll = time.Millisecond
	*sdk.SelectBudget = 50 * time.Millisecond

	s.T().Cleanup(func() {
		*sdk.SelectPoll, *sdk.SelectBudget = poll, budget
	})
}

// status is the device answering with one status and nothing else.
func (s *SelectTestSuite) status(txn uint64, status int) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(int64(status)))

	return sdk.Reply(sdk.DataChannel, buf.Bytes())
}

// document is the device answering with a preset.
func (s *SelectTestSuite) document(txn uint64, body string) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.EncodeString(body))

	return sdk.Reply(sdk.DataChannel, buf.Bytes())
}

// took is the device saying it has taken a request.
func (s *SelectTestSuite) took(txn uint64) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(1))

	return sdk.Reply(sdk.DataChannel, buf.Bytes())
}

// playing is the device saying which preset it has loaded.
func (s *SelectTestSuite) playing(txn uint64, setlist, slot int) []byte {
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

	return sdk.Reply(sdk.DataChannel, buf.Bytes())
}

func (s *SelectTestSuite) session(d *device) *sdk.Session {
	out := sdk.NewTestSession(d, d)
	out.OpenChannels()

	return out
}

// TestSelectPreset loads a preset and waits for the device to say it landed.
func (s *SelectTestSuite) TestSelectPreset() {
	tests := []struct {
		name      string
		device    func() *device
		cancelled bool
		says      string
	}{
		{
			// The device reports the preset it was playing before answering
			// with the one that was asked for, which is the window a caller
			// must not return inside.
			name: "one that answers with the old preset first",
			device: func() *device {
				return answers(
					s.took(sdk.FirstTxn),
					s.playing(sdk.FirstTxn+1, 0, 5),
					s.playing(sdk.FirstTxn+2, 0, 99),
				)
			},
		},
		{
			name: "one that takes it and never gets there",
			device: func() *device {
				return answers(s.took(sdk.FirstTxn), s.playing(sdk.FirstTxn+1, 0, 5))
			},
			says: "did not finish switching",
		},
		{
			name:   "one that never takes it at all",
			device: func() *device { return answers() },
			says:   "no reply",
		},
		{
			name: "an answer the protocol does not describe",
			device: func() *device {
				return answers(s.status(sdk.FirstTxn, 7))
			},
			says: "unexpected status 7",
		},
		{
			name: "a caller that gave up waiting",
			device: func() *device {
				return answers(s.took(sdk.FirstTxn), s.playing(sdk.FirstTxn+1, 0, 5))
			},
			cancelled: true,
			says:      "context canceled",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
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
func (s *SelectTestSuite) TestLoaded() {
	tests := []struct {
		name   string
		device func() *device
		want   wire.Loaded
		err    bool
	}{
		{
			name: "a device that says",
			device: func() *device {
				return answers(s.playing(sdk.FirstTxn, 0, 99))
			},
			want: wire.Loaded{Setlist: 0, Slot: 99, Name: "Chunky Monkey"},
		},
		{
			name:   "one that will not",
			device: func() *device { return answers() },
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
func (s *SelectTestSuite) TestReadCurrent() {
	tests := []struct {
		name   string
		device func() *device
		want   any
		err    bool
	}{
		{
			name: "a device holding a preset",
			device: func() *device {
				return answers(s.document(sdk.FirstTxn, "a preset"))
			},
			want: "a preset",
		},
		{
			name:   "one that will not answer",
			device: func() *device { return answers() },
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
	suite.Run(t, new(SelectTestSuite))
}
