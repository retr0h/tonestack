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

// TestOpenOver finds a device on a bus, claims it and hands back a session.
func (s *DiscoverBusTestSuite) TestOpenOver() {
	tests := []struct {
		name string
		bus  func() *fakeBus
		err  bool
		says string
		is   error
	}{
		{
			name: "a bus with one this package recognises",
			bus: func() *fakeBus {
				return &fakeBus{devices: []sdk.TestHandle{foreign(), helix(answers())}}
			},
		},
		{
			// Enumerating can fail partway and still have found something.
			// What it found is worth using.
			name: "one that complained and found something anyway",
			bus: func() *fakeBus {
				return &fakeBus{
					devices: []sdk.TestHandle{helix(answers())},
					err:     errors.New("boom"),
				}
			},
		},
		{
			name: "a bus with nothing on it but somebody else's device",
			bus: func() *fakeBus {
				return &fakeBus{devices: []sdk.TestHandle{foreign()}}
			},
			err: true,
			is:  sdk.ErrNoDevice,
		},
		{
			name: "one that cannot be looked at at all",
			bus:  func() *fakeBus { return &fakeBus{err: errors.New("boom")} },
			err:  true,
			says: "looking for a device",
		},
		{
			name: "an interface something else is holding",
			bus: func() *fakeBus {
				dev := helix(answers())
				dev.claimErr = errors.New("busy")

				return &fakeBus{devices: []sdk.TestHandle{dev}}
			},
			err:  true,
			says: "is HX Edit running?",
		},
		{
			// The interface is claimed twice, and the second can fail where
			// the first did not.
			name: "one that comes free and then does not",
			bus: func() *fakeBus {
				dev := helix(answers())
				dev.claimErr, dev.failClaim = errors.New("busy"), 1

				return &fakeBus{devices: []sdk.TestHandle{dev}}
			},
			err: true,
		},
		{
			name: "an outgoing endpoint that will not open",
			bus: func() *fakeBus {
				dev := helix(answers())
				dev.outErr = errors.New("boom")

				return &fakeBus{devices: []sdk.TestHandle{dev}}
			},
			err:  true,
			says: "outgoing",
		},
		{
			name: "an incoming one",
			bus: func() *fakeBus {
				dev := helix(answers())
				dev.inErr = errors.New("boom")

				return &fakeBus{devices: []sdk.TestHandle{dev}}
			},
			err:  true,
			says: "incoming",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.OpenOver(context.Background(), tt.bus())

			if !tt.err {
				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", got.Model().Name)

				return
			}

			s.Require().Error(err)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}

			if tt.says != "" {
				s.Require().Contains(err.Error(), tt.says)
			}
		})
	}
}

// TestOpenOverClaimsTwice is what HX Edit does, and what the device needs.
//
// The device carries channel state across connections, and the release
// between the two claims is what clears it.
func (s *DiscoverBusTestSuite) TestOpenOverClaimsTwice() {
	dev := helix(answers(
		sdk.FrameFor("control", wire.MsgHello, nil),
		sdk.FrameFor("control", wire.MsgAck, nil),
	))

	_, err := sdk.OpenOver(context.Background(),
		&fakeBus{devices: []sdk.TestHandle{dev}})

	s.Require().NoError(err)
	s.Require().Equal(2, dev.claims)
	s.Require().Equal(1, dev.released)
}

// TestOpenOverGivesBackWhatItDoesNotUse covers every device it looked at and
// did not want. One held by a process that is not using it is a device
// nothing else can claim.
func (s *DiscoverBusTestSuite) TestOpenOverGivesBackWhatItDoesNotUse() {
	first, second := helix(answers()), helix(answers())

	_, err := sdk.OpenOver(context.Background(),
		&fakeBus{devices: []sdk.TestHandle{first, second}})

	s.Require().NoError(err)
	s.Require().False(first.closed)
	s.Require().True(second.closed, "the one nobody used is given back")

	// And a bus nobody is using at all.
	empty := &fakeBus{devices: []sdk.TestHandle{foreign()}}

	_, err = sdk.OpenOver(context.Background(), empty)

	s.Require().Error(err)
	s.Require().True(empty.closed)
}

// TestOpenOverGivesBackAHalfBuiltSession covers a device that stops listening
// partway through opening its channels.
func (s *DiscoverBusTestSuite) TestOpenOverGivesBackAHalfBuiltSession() {
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

// TestOpenFindsItsOwnBus covers the one line in this package that reaches
// hardware.
func (s *DiscoverBusTestSuite) TestOpenFindsItsOwnBus() {
	restore := *sdk.NewBus
	defer func() { *sdk.NewBus = restore }()

	*sdk.NewBus = func() sdk.TestBus {
		return &fakeBus{devices: []sdk.TestHandle{helix(answers())}}
	}

	got, err := sdk.Open(context.Background())

	s.Require().NoError(err)
	s.Require().NotNil(got)
}

func TestDiscoverBusTestSuite(t *testing.T) {
	suite.Run(t, new(DiscoverBusTestSuite))
}
