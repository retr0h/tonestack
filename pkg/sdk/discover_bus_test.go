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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// DiscoverBusTestSuite covers finding a device and claiming it.
//
// All of it against a bus this file supplies. What libusb does is one
// expression per method in usb.go, and none of the decisions are there.
type DiscoverBusTestSuite struct {
	suite.Suite
}

// bus is a set of devices, or a failure to look for them.
type fakeBus struct {
	devices []sdk.TestHandle
	err     error
	closed  bool
}

func (b *fakeBus) Devices(
	match func(vendor, product uint16) bool,
) ([]sdk.TestHandle, error) {
	var out []sdk.TestHandle

	for _, d := range b.devices {
		desc := d.Descriptor()
		if match(desc.Vendor, desc.Product) {
			out = append(out, d)
		}
	}

	return out, b.err
}

func (b *fakeBus) Close() error {
	b.closed = true

	return nil
}

// handle is one device on it.
type fakeHandle struct {
	desc      sdk.Descriptor
	claimErr  error
	outErr    error
	inErr     error
	claims    int
	closed    bool
	released  int
	scripted  *device
	failClaim int
}

func (h *fakeHandle) Descriptor() sdk.Descriptor { return h.desc }

func (h *fakeHandle) Close() error {
	h.closed = true

	return nil
}

func (h *fakeHandle) Claim() (sdk.TestEndpoints, func(), error) {
	h.claims++

	if h.claimErr != nil && h.claims > h.failClaim {
		return nil, nil, h.claimErr
	}

	return &fakeEnds{h: h}, func() { h.released++ }, nil
}

// ends is a claimed interface.
type fakeEnds struct {
	h *fakeHandle
}

func (e *fakeEnds) Out() (sdk.TestSender, error) {
	if e.h.outErr != nil {
		return nil, e.h.outErr
	}

	return e.h.scripted, nil
}

func (e *fakeEnds) In() (sdk.TestReceiver, error) {
	if e.h.inErr != nil {
		return nil, e.h.inErr
	}

	return e.h.scripted, nil
}

// helix is a device this package recognises.
func helix(d *device) *fakeHandle {
	return &fakeHandle{
		desc:     sdk.Descriptor{Vendor: 0x0e41, Product: 0x4246},
		scripted: d,
	}
}

// foreign is a device on the bus that is nothing to do with this.
func foreign() *fakeHandle {
	return &fakeHandle{desc: sdk.Descriptor{Vendor: 0x1234, Product: 0x5678}}
}

func (s *DiscoverBusTestSuite) TestOpensTheFirstDeviceItRecognises() {
	d := answers(
		sdk.FrameFor("control", wire.MsgHello, nil),
		sdk.FrameFor("control", wire.MsgAck, nil),
	)
	dev := helix(d)
	b := &fakeBus{devices: []sdk.TestHandle{foreign(), dev}}

	session, err := sdk.OpenOver(context.Background(), b)

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", session.Model().Name)

	// Claimed, released, and claimed again, which is what HX Edit does: the
	// device carries channel state across connections and the release is what
	// clears it.
	s.Require().Equal(2, dev.claims)
	s.Require().Equal(1, dev.released)
}

func (s *DiscoverBusTestSuite) TestClosesEveryDeviceItDoesNotUse() {
	// A device held by a process that is not using it is a device nothing
	// else can claim.
	first, second := helix(answers()), helix(answers())
	b := &fakeBus{devices: []sdk.TestHandle{first, second}}

	_, err := sdk.OpenOver(context.Background(), b)

	s.Require().NoError(err)
	s.Require().False(first.closed)
	s.Require().True(second.closed, "the one nobody used is given back")
}

func (s *DiscoverBusTestSuite) TestReportsNothingAttached() {
	b := &fakeBus{devices: []sdk.TestHandle{foreign()}}

	_, err := sdk.OpenOver(context.Background(), b)

	s.Require().ErrorIs(err, sdk.ErrNoDevice)
	s.Require().True(b.closed, "a bus nobody is using is given back")
}

func (s *DiscoverBusTestSuite) TestReportsABusItCannotLookAt() {
	b := &fakeBus{err: errors.New("boom")}

	_, err := sdk.OpenOver(context.Background(), b)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "looking for a device")
}

func (s *DiscoverBusTestSuite) TestADeviceFoundDespiteAComplaint() {
	// Enumerating can fail partway and still have found something. What it
	// found is worth using.
	b := &fakeBus{devices: []sdk.TestHandle{helix(answers())}, err: errors.New("boom")}

	_, err := sdk.OpenOver(context.Background(), b)

	s.Require().NoError(err)
}

func (s *DiscoverBusTestSuite) TestReportsAnInterfaceItCannotClaim() {
	dev := helix(answers())
	dev.claimErr = errors.New("busy")

	_, err := sdk.OpenOver(context.Background(),
		&fakeBus{devices: []sdk.TestHandle{dev}})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "is HX Edit running?")
}

func (s *DiscoverBusTestSuite) TestReportsASecondClaimItCannotMake() {
	dev := helix(answers())
	dev.claimErr, dev.failClaim = errors.New("busy"), 1

	_, err := sdk.OpenOver(context.Background(),
		&fakeBus{devices: []sdk.TestHandle{dev}})

	s.Require().Error(err)
}

func (s *DiscoverBusTestSuite) TestReportsAnEndpointItCannotOpen() {
	for _, tc := range []struct {
		name string
		set  func(*fakeHandle)
		want string
	}{
		{"outgoing", func(h *fakeHandle) { h.outErr = errors.New("boom") }, "outgoing"},
		{"incoming", func(h *fakeHandle) { h.inErr = errors.New("boom") }, "incoming"},
	} {
		s.Run(tc.name, func() {
			dev := helix(answers())
			tc.set(dev)

			_, err := sdk.OpenOver(context.Background(),
				&fakeBus{devices: []sdk.TestHandle{dev}})

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.want)
		})
	}
}

func (s *DiscoverBusTestSuite) TestReportsAHandshakeThatFails() {
	// A device that stops listening partway through opening its channels
	// leaves the session half built, and what was taken is given back.
	d := answers()
	d.writeErr = errors.New("boom")

	dev := helix(d)

	_, err := sdk.OpenOver(context.Background(),
		&fakeBus{devices: []sdk.TestHandle{dev}})

	s.Require().Error(err)

	// Twice: once between the two claims, and once when the session that
	// could not finish gives back what it took.
	s.Require().Equal(2, dev.released, "the interface is given back")
}

func (s *DiscoverBusTestSuite) TestOpenFindsItsOwnBus() {
	// The one line in this package that reaches hardware.
	restore := *sdk.NewBus
	defer func() { *sdk.NewBus = restore }()

	*sdk.NewBus = func() sdk.TestBus {
		return &fakeBus{devices: []sdk.TestHandle{helix(answers())}}
	}

	session, err := sdk.Open(context.Background())

	s.Require().NoError(err)
	s.Require().NotNil(session)
}

func TestDiscoverBusTestSuite(t *testing.T) {
	suite.Run(t, new(DiscoverBusTestSuite))
}
