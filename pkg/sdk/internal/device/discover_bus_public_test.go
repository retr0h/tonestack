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
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// DiscoverBusPublicTestSuite covers finding a device and claiming it.
//
// All of it against a bus this file supplies. What libusb does is one
// expression per method in usb_darwin.go, and none of the decisions are there.
type DiscoverBusPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *DiscoverBusPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// busDouble is a set of devices, or a failure to look for them: a Mockbus
// wired to the knobs a test sets directly, the same way handleDouble wires a
// Mockhandle.
type busDouble struct {
	mock    *device.Mockbus
	handles []*handleDouble
	err     error
	closed  bool
}

// bus wires a Mockbus to a set of devices, or to a failure to look for them.
func (s *DiscoverBusPublicTestSuite) bus(
	err error,
	handles ...*handleDouble,
) *busDouble {
	b := &busDouble{handles: handles, err: err}

	b.mock = device.NewMockbus(s.ctrl)
	b.mock.EXPECT().Devices(gomock.Any()).DoAndReturn(
		func(match func(vendor, product uint16) bool) ([]device.TestHandle, error) {
			var out []device.TestHandle

			for _, h := range b.handles {
				if match(h.desc.Vendor, h.desc.Product) {
					out = append(out, h.mock)
				}
			}

			return out, b.err
		}).AnyTimes()
	b.mock.EXPECT().Close().DoAndReturn(func() error {
		b.closed = true

		return nil
	}).AnyTimes()

	return b
}

// handleDouble is one device on a fake bus: a Mockhandle wired to the knobs
// a test sets directly, the same way deviceDouble wires a sender and a
// receiver.
type handleDouble struct {
	mock *device.Mockhandle

	desc      device.Descriptor
	claimErr  error
	outErr    error
	inErr     error
	claims    int
	closed    bool
	released  int
	device    *deviceDouble
	failClaim int
}

// handle wires a Mockhandle to a device.
func (s *DiscoverBusPublicTestSuite) handle(
	desc device.Descriptor,
	d *deviceDouble,
) *handleDouble {
	h := &handleDouble{desc: desc, device: d}

	h.mock = device.NewMockhandle(s.ctrl)
	h.mock.EXPECT().Descriptor().DoAndReturn(func() device.Descriptor {
		return h.desc
	}).AnyTimes()
	h.mock.EXPECT().Close().DoAndReturn(func() error {
		h.closed = true

		return nil
	}).AnyTimes()
	h.mock.EXPECT().Claim().DoAndReturn(func() (device.TestEndpoints, func(), error) {
		h.claims++

		if h.claimErr != nil && h.claims > h.failClaim {
			return nil, nil, h.claimErr
		}

		ends := device.NewMockendpoints(s.ctrl)
		ends.EXPECT().Out().DoAndReturn(func() (device.TestSender, error) {
			if h.outErr != nil {
				return nil, h.outErr
			}

			return h.device.out, nil
		}).AnyTimes()
		ends.EXPECT().In().DoAndReturn(func() (device.TestReceiver, error) {
			if h.inErr != nil {
				return nil, h.inErr
			}

			return h.device.in, nil
		}).AnyTimes()

		return ends, func() { h.released++ }, nil
	}).AnyTimes()

	return h
}

// helix is a device this package recognises.
func (s *DiscoverBusPublicTestSuite) helix(
	d *deviceDouble,
) *handleDouble {
	return s.handle(device.Descriptor{Vendor: 0x0e41, Product: 0x4246}, d)
}

// foreign is a device on the bus that is nothing to do with this.
func (s *DiscoverBusPublicTestSuite) foreign() *handleDouble {
	return s.handle(device.Descriptor{Vendor: 0x1234, Product: 0x5678}, nil)
}

// TestOpenOver finds a device on a bus, claims it and hands back a session.
func (s *DiscoverBusPublicTestSuite) TestOpenOver() {
	tests := []struct {
		name string
		bus  func() *busDouble
		// how many times the interface must be claimed and given back, and
		// which handles must be closed.
		claims   int
		released int
		// every device it looked at and did not want must be given back: one
		// held by a process that is not using it is a device nothing else can
		// claim.
		unusedClosed bool
		busClosed    bool
		// every handle the bus returned was given back, and the first was
		// never claimed.
		allClosed bool
		unclaimed bool

		err  bool
		says string
		is   error
	}{
		{
			name: "a bus with one this package recognises",
			bus: func() *busDouble {
				return s.bus(nil, s.foreign(), s.helix(answers(s.ctrl)))
			},
		},
		{
			// Enumerating can fail partway and still have found something.
			// A listing that failed is not one to claim hardware from, and
			// nothing it returned is left held.
			name: "one that complained and found something anyway",
			bus: func() *busDouble {
				return s.bus(errors.New("boom"),
					s.helix(answers(s.ctrl)), s.helix(answers(s.ctrl)))
			},
			err:       true,
			says:      "looking for a device",
			busClosed: true,
			allClosed: true,
			unclaimed: true,
		},
		{
			// A product identifier is only Line 6's under Line 6's vendor
			// identifier. Somebody else's device that happens to share one is
			// not a Helix, and claiming it talks this protocol at hardware
			// that does not speak it.
			name: "a Helix product identifier under somebody else's vendor",
			bus: func() *busDouble {
				return s.bus(nil, s.handle(
					device.Descriptor{Vendor: 0x1234, Product: 0x4246}, answers(s.ctrl)))
			},
			err:       true,
			is:        device.ErrNoDevice,
			busClosed: true,
			unclaimed: true,
		},
		{
			name: "a device whose reads fail before the handshake starts",
			bus: func() *busDouble {
				return s.bus(nil, s.helix(readFails(s.ctrl, errors.New("boom"))))
			},
			err:  true,
			says: "reading from the device",
		},
		{
			// The drain reads a quiet device; the read that fails is the
			// one waiting on an answer to the channel's opening frame.
			name: "one whose reads fail after the opening frame",
			bus: func() *busDouble {
				return s.bus(nil, s.helix(readFailsAfter(s.ctrl, errors.New("boom"), 1)))
			},
			err:  true,
			says: "reading from the device",
		},
		{
			name: "one whose reads fail after a service is asked for",
			bus: func() *busDouble {
				return s.bus(nil, s.helix(readFailsAfter(s.ctrl, errors.New("boom"), 2)))
			},
			err:  true,
			says: "reading from the device",
		},
		{
			// The wait between closing a channel and reopening it: the
			// opening, the service, its acknowledgement, then the close.
			name: "one whose reads fail after a channel is closed",
			bus: func() *busDouble {
				return s.bus(nil, s.helix(readFailsAfter(s.ctrl, errors.New("boom"), 4)))
			},
			err:  true,
			says: "reading from the device",
		},
		{
			name: "a bus with nothing on it but somebody else's device",
			bus: func() *busDouble {
				return s.bus(nil, s.foreign())
			},
			busClosed: true,
			err:       true,
			is:        device.ErrNoDevice,
		},
		{
			// The device carries channel state across connections, and the
			// release between the two claims is what clears it. This is what
			// HX Edit does, and what the device needs.
			name: "a device claimed twice, as the device requires",
			bus: func() *busDouble {
				return s.bus(nil, s.helix(answers(s.ctrl,
					device.FrameFor("control", wire.MsgHello, nil),
					device.FrameFor("control", wire.MsgAck, nil),
				)))
			},
			claims:   2,
			released: 1,
		},
		{
			name: "a bus carrying more than one of them",
			bus: func() *busDouble {
				return s.bus(nil, s.helix(answers(s.ctrl)), s.helix(answers(s.ctrl)))
			},
			unusedClosed: true,
		},
		{
			name: "a device that stops listening partway through opening",
			bus: func() *busDouble {
				d := answers(s.ctrl)
				d.writeErr = errors.New("boom")

				return s.bus(nil, s.helix(d))
			},
			// Twice: once between the two claims, and once when the session
			// that could not finish gives back what it took.
			released: 2,
			err:      true,
		},
		{
			name: "one that cannot be looked at at all",
			bus:  func() *busDouble { return s.bus(errors.New("boom")) },
			err:  true,
			says: "looking for a device",
		},
		{
			name: "an interface something else is holding",
			bus: func() *busDouble {
				h := s.helix(answers(s.ctrl))
				h.claimErr = errors.New("busy")

				return s.bus(nil, h)
			},
			err:  true,
			says: "is HX Edit running?",
		},
		{
			// The interface is claimed twice, and the second can fail where
			// the first did not.
			name: "one that comes free and then does not",
			bus: func() *busDouble {
				h := s.helix(answers(s.ctrl))
				h.claimErr, h.failClaim = errors.New("busy"), 1

				return s.bus(nil, h)
			},
			err: true,
		},
		{
			name: "an outgoing endpoint that will not open",
			bus: func() *busDouble {
				h := s.helix(answers(s.ctrl))
				h.outErr = errors.New("boom")

				return s.bus(nil, h)
			},
			err:  true,
			says: "outgoing",
		},
		{
			name: "an incoming one",
			bus: func() *busDouble {
				h := s.helix(answers(s.ctrl))
				h.inErr = errors.New("boom")

				return s.bus(nil, h)
			},
			err:  true,
			says: "incoming",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			b := tt.bus()

			got, err := device.OpenOver(context.Background(), b.mock)
			if got != nil {
				// After the assertions below, which count what the open
				// itself gave back.
				s.T().Cleanup(func() { _ = got.Close() })
			}

			if tt.err {
				s.Require().Error(err)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				if tt.says != "" {
					s.Require().Contains(err.Error(), tt.says)
				}
			} else {
				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", got.Model().Name)
			}

			var first *handleDouble
			if len(b.handles) > 0 {
				first = b.handles[0]
			}

			if tt.claims > 0 {
				s.Require().Equal(tt.claims, first.claims)
			}

			if tt.released > 0 {
				s.Require().Equal(tt.released, first.released,
					"the interface is given back")
			}

			if tt.unusedClosed {
				second := b.handles[1]

				s.Require().False(first.closed)
				s.Require().True(second.closed, "the one nobody used is given back")
			}

			if tt.busClosed {
				s.Require().True(b.closed, "a bus nobody is using is given back")
			}

			if tt.unclaimed {
				s.Require().Zero(first.claims, "nothing is claimed")
			}

			if tt.allClosed {
				for i, h := range b.handles {
					s.Require().True(h.closed, "handle %d is given back", i)
				}
			}
		})
	}
}

// TestOpenFindsItsOwnBus covers the one line in this package that reaches
// hardware.
func (s *DiscoverBusPublicTestSuite) TestOpenFindsItsOwnBus() {
	source := device.NewMockbuses(s.ctrl)
	source.EXPECT().Bus().Return(s.bus(nil, s.helix(answers(s.ctrl))).mock)

	var trace bytes.Buffer

	got, err := device.NewUSBOver(&trace, source).Open(context.Background())

	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().NoError(got.Close())

	// The trace the opener was given reaches the session it opens: the
	// handshake goes out through it.
	s.Require().Contains(trace.String(), "OUT ")
}

// TestUSBBus covers where NewUSB gets its bus. Opening one reads nothing;
// only looking for devices on it does.
func (s *DiscoverBusPublicTestSuite) TestUSBBus() {
	b := device.USBBus()

	s.Require().NotNil(b)
	s.Require().NoError(b.Close())
}

func TestDiscoverBusPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DiscoverBusPublicTestSuite))
}
