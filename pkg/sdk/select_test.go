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

func (s *SelectTestSuite) TestWaitsUntilTheDeviceSaysItLanded() {
	// The device reports the preset it was playing before answering with the
	// one that was asked for, which is the window a caller must not return
	// inside.
	d := answers(
		s.took(sdk.FirstTxn),
		s.playing(sdk.FirstTxn+1, 0, 5),
		s.playing(sdk.FirstTxn+2, 0, 99),
	)

	s.Require().NoError(s.session(d).SelectPreset(context.Background(), 0, 99))
	s.Require().Empty(d.replies, "every answer was read")
}

func (s *SelectTestSuite) TestReportsASwitchThatNeverLands() {
	d := answers(
		s.took(sdk.FirstTxn),
		s.playing(sdk.FirstTxn+1, 0, 5),
	)

	err := s.session(d).SelectPreset(context.Background(), 0, 99)

	s.Require().ErrorContains(err, "did not finish switching")
}

func (s *SelectTestSuite) TestReportsADeviceThatWillNotTakeIt() {
	err := s.session(answers()).SelectPreset(context.Background(), 0, 99)

	s.Require().Error(err)
}

func (s *SelectTestSuite) TestGivesUpWhenTheCallerDoes() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := answers(s.took(sdk.FirstTxn), s.playing(sdk.FirstTxn+1, 0, 5))

	s.Require().Error(s.session(d).SelectPreset(ctx, 0, 99))
}

// TestReportsAnAnswerNobodyExpects covers a status that is neither "taken"
// nor "done". A refusal is already an error by the time it arrives here, so
// this is a device saying something the protocol does not describe.
func (s *SelectTestSuite) TestReportsAnAnswerNobodyExpects() {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(sdk.FirstTxn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(7))

	d := answers(sdk.Reply(sdk.DataChannel, buf.Bytes()))

	err := s.session(d).SelectPreset(context.Background(), 0, 99)

	s.Require().ErrorContains(err, "unexpected status 7")
}

// TestReadsTheDocumentBeingPlayed covers the edit buffer, which is what a
// device is making a sound with rather than what it has stored.
func (s *SelectTestSuite) TestReadsTheDocumentBeingPlayed() {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(sdk.FirstTxn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.EncodeString("a preset"))

	got, err := s.session(answers(sdk.Reply(sdk.DataChannel, buf.Bytes()))).
		ReadCurrent(context.Background())

	s.Require().NoError(err)
	s.Require().Equal("a preset", got)
}

// TestReportsADeviceThatWillNotSayWhatItPlays covers silence.
func (s *SelectTestSuite) TestReportsADeviceThatWillNotSayWhatItPlays() {
	_, err := s.session(answers()).ReadCurrent(context.Background())

	s.Require().Error(err)
}

func (s *SelectTestSuite) TestReadsWhatTheDeviceIsPlaying() {
	d := answers(s.playing(sdk.FirstTxn, 0, 99))

	got, err := s.session(d).Loaded(context.Background())

	s.Require().NoError(err)
	s.Require().Equal(0, got.Setlist)
	s.Require().Equal(99, got.Slot)
	s.Require().Equal("Chunky Monkey", got.Name)
}

func TestSelectTestSuite(t *testing.T) {
	suite.Run(t, new(SelectTestSuite))
}
