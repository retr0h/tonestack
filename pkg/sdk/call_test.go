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

// session returns one with its channels already open, over a scripted device.
func (s *CallTestSuite) session(d *device) *sdk.Session {
	out := sdk.NewTestSession(d, d)
	out.OpenChannels()

	return out
}

func (s *CallTestSuite) TestACallGetsItsAnswer() {
	d := answers(sdk.Reply(sdk.ControlChannel, s.answer(sdk.FirstTxn, 0, "done")))

	got, err := s.session(d).Call(
		context.Background(), sdk.ControlChannel, 1, nil)

	s.Require().NoError(err)
	s.Require().Equal(uint64(sdk.FirstTxn), got.Txn)
	s.Require().Equal("done", got.Result)
	s.Require().NotEmpty(d.sent, "a call is a request before it is an answer")
}

func (s *CallTestSuite) TestACallIgnoresSomebodyElsesAnswer() {
	// A notification carries no transaction and is not anybody's reply.
	// Letting one be mistaken for this reply would answer the wrong question.
	d := answers(
		sdk.Reply(sdk.ControlChannel, s.answer(sdk.FirstTxn+99, 0, "not yours")),
		sdk.Reply(sdk.ControlChannel, s.answer(sdk.FirstTxn, 0, "yours")),
	)

	got, err := s.session(d).Call(
		context.Background(), sdk.ControlChannel, 1, nil)

	s.Require().NoError(err)
	s.Require().Equal("yours", got.Result)
}

func (s *CallTestSuite) TestACallOnAChannelNobodyOpened() {
	d := answers()

	_, err := sdk.NewTestSession(d, d).Call(
		context.Background(), "nowhere", 1, nil)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "no nowhere channel")
}

func (s *CallTestSuite) TestACallOnABusItCannotWriteTo() {
	d := &device{writeErr: errors.New("boom")}

	_, err := s.session(d).Call(context.Background(), sdk.ControlChannel, 1, nil)

	s.Require().Error(err)
}

func (s *CallTestSuite) TestADeviceThatRefuses() {
	// Status 255 is a refusal, and the code it carries is the useful half.
	d := answers(sdk.Reply(sdk.ControlChannel,
		s.answer(sdk.FirstTxn, 255, map[int]int{111: 7})))

	_, err := s.session(d).Call(context.Background(), sdk.ControlChannel, 4, nil)

	s.Require().ErrorIs(err, wire.ErrRefused)
	s.Require().Contains(err.Error(), "opcode 4")
}

func (s *CallTestSuite) TestACallThatIsNeverAnswered() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.session(answers()).Call(ctx, sdk.ControlChannel, 1, nil)

	s.Require().Error(err)
}

func (s *CallTestSuite) TestListingPresets() {
	rows := []any{
		map[int]any{0: map[int]any{109: "Chunky Monkey\x00"}},
		map[int]any{1: map[int]any{109: "Fat Mike\x00"}},
	}
	d := answers(sdk.Reply(sdk.ControlChannel, s.answer(sdk.FirstTxn, 0, rows)))

	got, err := s.session(d).Presets(context.Background(), 0)

	s.Require().NoError(err)
	s.Require().Len(got, 2)

	// Read by position, not by the key each entry carries: that key is the
	// index a preset had before it was last reordered on the pedal.
	s.Require().Equal("Chunky Monkey", got[0].Name)
	s.Require().Equal(0, got[0].Slot)
	s.Require().Equal("Fat Mike", got[1].Name)
	s.Require().Equal(1, got[1].Slot)
}

func (s *CallTestSuite) TestReadingOnePreset() {
	d := answers(sdk.Reply(sdk.ControlChannel,
		s.answer(sdk.FirstTxn, 0, "a preset")))

	got, err := s.session(d).ReadPreset(context.Background(), 0, 3)

	s.Require().NoError(err)
	s.Require().Equal("a preset", got)
}

func (s *CallTestSuite) TestReportsAListingItCannotGet() {
	d := &device{writeErr: errors.New("boom")}

	_, err := s.session(d).Presets(context.Background(), 0)
	s.Require().Error(err)

	_, err = s.session(d).ReadPreset(context.Background(), 0, 0)
	s.Require().Error(err)
}

func (s *CallTestSuite) TestAnAnswerThatWillNotDecode() {
	// Whatever arrives is not guaranteed to be a reply. One that cannot be
	// read is skipped rather than mistaken for this call's answer.
	d := answers(
		sdk.Reply(sdk.ControlChannel, []byte{0xc1}),
		sdk.Reply(sdk.ControlChannel, s.answer(sdk.FirstTxn, 0, "yours")),
	)

	got, err := s.session(d).Call(context.Background(), sdk.ControlChannel, 1, nil)

	s.Require().NoError(err)
	s.Require().Equal("yours", got.Result)
}

func (s *CallTestSuite) TestAnAcknowledgementTheBusRefuses() {
	// Bytes that arrived are acknowledged, and a device that stops listening
	// at that point has to be reported: an unacknowledged stream stalls.
	d := answers(sdk.Reply(sdk.ControlChannel, []byte{0xc1}))
	out := &sdk.FailAfter{Sender: d, OK: 1, Err: errors.New("boom")}

	session := sdk.NewTestSession(out, d)
	session.OpenChannels()

	_, err := session.Call(context.Background(), sdk.ControlChannel, 1, nil)

	s.Require().Error(err)
}

func TestCallTestSuite(t *testing.T) {
	suite.Run(t, new(CallTestSuite))
}
