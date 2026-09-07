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
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// TransportTestSuite exercises the protocol against a scripted device.
//
// Everything a session does apart from finding and claiming hardware is here:
// framing, sequence numbers, acknowledgements, opening a channel and making a
// call. None of it needs a device, and all of it is what breaks one when it is
// wrong.
type TransportTestSuite struct {
	suite.Suite
}

// device is a scripted answer to whatever is written to it.
type device struct {
	// replies are handed back one read at a time.
	replies [][]byte
	// sent is everything the session wrote.
	sent [][]byte
	// writeErr fails every write.
	writeErr error
	// readErr fails every read.
	readErr error
}

func (d *device) Write(p []byte) (int, error) {
	if d.writeErr != nil {
		return 0, d.writeErr
	}

	d.sent = append(d.sent, append([]byte(nil), p...))

	return len(p), nil
}

func (d *device) ReadContext(_ context.Context, p []byte) (int, error) {
	if d.readErr != nil {
		return 0, d.readErr
	}

	if len(d.replies) == 0 {
		// A device with nothing to say answers with nothing, which is how a
		// drain knows it has finished.
		return 0, nil
	}

	reply := d.replies[0]
	d.replies = d.replies[1:]

	return copy(p, reply), nil
}

// answers scripts a device to reply with each of the given frames in turn.
func answers(frames ...[]byte) *device { return &device{replies: frames} }

func (s *TransportTestSuite) TestADrainFinishesWhenTheDeviceGoesQuiet() {
	d := answers()

	sdk.NewTestSession(d, d).Drain(context.Background())
}

func (s *TransportTestSuite) TestABusThatWillNotAnswerReadsAsQuiet() {
	// A read that fails is treated as the device having nothing to say,
	// because that is what a timeout looks like and a timeout is ordinary.
	// The cost is that a genuine bus failure surfaces later, as a call with
	// no reply, rather than here.
	for _, boom := range []error{errors.New("boom"), io.EOF} {
		d := &device{readErr: boom}

		sdk.NewTestSession(d, d).Drain(context.Background())
	}
}

func (s *TransportTestSuite) TestADrainConsumesWhatArrives() {
	// A drain that saw traffic starts counting quiet reads again, and what
	// it consumed is not replayed into a later reply.
	d := answers(sdk.FrameFor("control", wire.MsgData, []byte("noise")))

	session := sdk.NewTestSession(d, d)
	session.OpenChannels()
	session.Drain(context.Background())

	s.Require().Empty(d.replies, "everything the device had was read")
}

func (s *TransportTestSuite) TestAFrameForNoChannelAnybodyOpened() {
	// The device sends notifications unasked, and one on a channel this
	// session never opened is not anybody's business.
	d := answers(sdk.FrameFor("events", wire.MsgData, []byte("noise")))

	s.Require().False(sdk.NewTestSession(d, d).Receive(context.Background()),
		"nothing arrived that anybody is waiting for")
}

func (s *TransportTestSuite) TestAPartialFrameEndsTheTransfer() {
	// A frame cut short at the end of a transfer is the transfer ending, not
	// corruption.
	full := sdk.FrameFor("control", wire.MsgData, []byte("noise"))
	d := answers(full[:6])

	session := sdk.NewTestSession(d, d)
	session.OpenChannels()

	s.Require().False(session.Receive(context.Background()))
}

func (s *TransportTestSuite) TestTheWireTrace() {
	// How both directions were read off a device in the first place.
	defer sdk.SetDebug(true)()

	d := answers(sdk.FrameFor("control", wire.MsgData, []byte("noise")))

	session := sdk.NewTestSession(d, d)
	session.OpenChannels()
	session.Drain(context.Background())

	// Both directions: what was asked as well as what came back.
	_, _ = session.Call(context.Background(), sdk.ControlChannel, 1, nil)
}

func (s *TransportTestSuite) TestAHandshakeOpensEveryChannel() {
	// Three channels, and the control channel twice — it serves two services
	// and the first is closed before the second is opened. Multiplexing them
	// onto one open channel silently breaks every channel.
	d := answers(
		sdk.FrameFor("control", wire.MsgHello, nil),
		sdk.FrameFor("control", wire.MsgAck, nil),
		sdk.FrameFor("control", wire.MsgHello, nil),
		sdk.FrameFor("events", wire.MsgHello, nil),
		sdk.FrameFor("data", wire.MsgHello, nil),
	)

	err := sdk.NewTestSession(d, d).Handshake(context.Background())

	s.Require().NoError(err)
	s.Require().NotEmpty(d.sent, "a handshake is what the session says first")
}

func (s *TransportTestSuite) TestAHandshakeReportsABusItCannotWriteTo() {
	d := &device{writeErr: errors.New("boom")}

	s.Require().Error(sdk.NewTestSession(d, d).Handshake(context.Background()))
}

func (s *TransportTestSuite) TestAHandshakeReportsAChannelItCannotClose() {
	// The control channel serves two services, so the first is closed before
	// the second is opened. A device that stops listening partway through
	// leaves it half open, and saying so beats carrying on.
	d := answers()
	out := &sdk.FailAfter{Sender: d, OK: 1, Err: errors.New("boom")}

	s.Require().Error(sdk.NewTestSession(out, d).Handshake(context.Background()))
}

func (s *TransportTestSuite) TestAHandshakeReportsAServiceItCannotOpen() {
	d := answers()
	out := &sdk.FailAfter{Sender: d, OK: 3, Err: errors.New("boom")}

	s.Require().Error(sdk.NewTestSession(out, d).Handshake(context.Background()))
}

func (s *TransportTestSuite) TestAHandshakeCarriesOnThroughSilence() {
	// A device that answers nothing is not an error here: opening a channel
	// writes and moves on, and what goes wrong shows up at the first call.
	d := &device{readErr: errors.New("boom")}

	s.Require().NoError(sdk.NewTestSession(d, d).Handshake(context.Background()))
}

func (s *TransportTestSuite) TestClosingGivesBackWhatItTook() {
	// What the device sent is drained and acknowledged first: dropping the
	// interface with bytes unacknowledged carries a debt into later sessions,
	// until an otherwise innocent write stops the device.
	d := answers(sdk.FrameFor("control", wire.MsgData, []byte("noise")))

	var given []string

	session := sdk.NewTestSession(d, d)
	session.OpenChannels()
	session.OnDone(func() { given = append(given, "interface") })
	session.Holding(
		func() error { given = append(given, "device"); return nil },
		func() error { given = append(given, "library"); return nil },
	)

	session.Close()

	// In the order they were taken.
	s.Require().Equal([]string{"interface", "device", "library"}, given)
	s.Require().NotEmpty(d.sent, "the acknowledgement is what settles the debt")
}

func (s *TransportTestSuite) TestASessionKnowsWhatAnswered() {
	s.Require().Equal("HX Stomp", sdk.NewTestSession(nil, nil).Model().Name)
}

func (s *TransportTestSuite) TestClosingASessionThatNeverOpened() {
	// Nothing was taken, so there is nothing to give back.
	sdk.NewTestSession(nil, nil).Close()
}

func (s *TransportTestSuite) TestWaitingOnABusyInterface() {
	// Cleanup after a previous session races the next claim, so an interface
	// that is busy is worth waiting on rather than reporting.
	tries := 0

	s.Require().NoError(sdk.Retry(func() error {
		tries++
		if tries < 2 {
			return errors.New("busy")
		}

		return nil
	}))

	s.Require().Equal(2, tries)
}

func (s *TransportTestSuite) TestAnInterfaceThatNeverComesFree() {
	tries := 0

	err := sdk.Retry(func() error {
		tries++

		return errors.New("busy")
	})

	s.Require().Error(err)
	s.Require().Equal(sdk.ClaimAttempts, tries, "patience runs out")
}

func TestTransportTestSuite(t *testing.T) {
	suite.Run(t, new(TransportTestSuite))
}
