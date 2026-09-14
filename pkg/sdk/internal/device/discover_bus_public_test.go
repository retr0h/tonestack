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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// DiscoverBusPublicTestSuite covers finding a device and claiming it.
//
// All of it against a bus this file supplies. What libusb does is one
// expression per method in usb_darwin.go, and none of the decisions are there.
type DiscoverBusPublicTestSuite struct {
	suite.Suite
}

// bus is a set of devices, or a failure to look for them.
type fakeBus struct {
	devices []device.TestHandle
	err     error
	closed  bool
}

func (b *fakeBus) Devices(
	match func(vendor, product uint16) bool,
) ([]device.TestHandle, error) {
	var out []device.TestHandle

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
	desc      device.Descriptor
	claimErr  error
	outErr    error
	inErr     error
	claims    int
	closed    bool
	released  int
	scripted  *scripted
	failClaim int
}

func (h *fakeHandle) Descriptor() device.Descriptor { return h.desc }

func (h *fakeHandle) Close() error {
	h.closed = true

	return nil
}

func (h *fakeHandle) Claim() (device.TestEndpoints, func(), error) {
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

func (e *fakeEnds) Out() (device.TestSender, error) {
	if e.h.outErr != nil {
		return nil, e.h.outErr
	}

	return e.h.scripted, nil
}

func (e *fakeEnds) In() (device.TestReceiver, error) {
	if e.h.inErr != nil {
		return nil, e.h.inErr
	}

	return e.h.scripted, nil
}

// helix is a device this package recognises.
func helix(d *scripted) *fakeHandle {
	return &fakeHandle{
		desc:     device.Descriptor{Vendor: 0x0e41, Product: 0x4246},
		scripted: d,
	}
}

// foreign is a device on the bus that is nothing to do with this.
func foreign() *fakeHandle {
	return &fakeHandle{desc: device.Descriptor{Vendor: 0x1234, Product: 0x5678}}
}

// TestOpenOver finds a device on a bus, claims it and hands back a session.
func (s *DiscoverBusPublicTestSuite) TestOpenOver() {
	tests := []struct {
		name string
		bus  func() *fakeBus
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
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{foreign(), helix(answers())}}
			},
		},
		{
			// Enumerating can fail partway and still have found something.
			// A listing that failed is not one to claim hardware from, and
			// nothing it returned is left held.
			name: "one that complained and found something anyway",
			bus: func() *fakeBus {
				return &fakeBus{
					devices: []device.TestHandle{helix(answers()), helix(answers())},
					err:     errors.New("boom"),
				}
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
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{&fakeHandle{
					desc:     device.Descriptor{Vendor: 0x1234, Product: 0x4246},
					scripted: answers(),
				}}}
			},
			err:       true,
			is:        device.ErrNoDevice,
			busClosed: true,
			unclaimed: true,
		},
		{
			name: "a device whose reads fail before the handshake starts",
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{helix(&scripted{
					readErr: errors.New("boom"),
				})}}
			},
			err:  true,
			says: "reading from the device",
		},
		{
			// Three quiet reads drain the device; the fourth is the first
			// read after a channel opening.
			name: "one whose reads fail after the opening frame",
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{helix(&scripted{
					readErr: errors.New("boom"), readsOK: 3,
				})}}
			},
			err:  true,
			says: "reading from the device",
		},
		{
			name: "one whose reads fail after a service is asked for",
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{helix(&scripted{
					readErr: errors.New("boom"), readsOK: 4,
				})}}
			},
			err:  true,
			says: "reading from the device",
		},
		{
			// The read between closing a channel and reopening it.
			name: "one whose reads fail after a channel is closed",
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{helix(&scripted{
					readErr: errors.New("boom"), readsOK: 5,
				})}}
			},
			err:  true,
			says: "reading from the device",
		},
		{
			name: "a bus with nothing on it but somebody else's device",
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{foreign()}}
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
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{helix(answers(
					device.FrameFor("control", wire.MsgHello, nil),
					device.FrameFor("control", wire.MsgAck, nil),
				))}}
			},
			claims:   2,
			released: 1,
		},
		{
			name: "a bus carrying more than one of them",
			bus: func() *fakeBus {
				return &fakeBus{devices: []device.TestHandle{
					helix(answers()), helix(answers()),
				}}
			},
			unusedClosed: true,
		},
		{
			name: "a device that stops listening partway through opening",
			bus: func() *fakeBus {
				d := answers()
				d.writeErr = errors.New("boom")

				return &fakeBus{devices: []device.TestHandle{helix(d)}}
			},
			// Twice: once between the two claims, and once when the session
			// that could not finish gives back what it took.
			released: 2,
			err:      true,
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

				return &fakeBus{devices: []device.TestHandle{dev}}
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

				return &fakeBus{devices: []device.TestHandle{dev}}
			},
			err: true,
		},
		{
			name: "an outgoing endpoint that will not open",
			bus: func() *fakeBus {
				dev := helix(answers())
				dev.outErr = errors.New("boom")

				return &fakeBus{devices: []device.TestHandle{dev}}
			},
			err:  true,
			says: "outgoing",
		},
		{
			name: "an incoming one",
			bus: func() *fakeBus {
				dev := helix(answers())
				dev.inErr = errors.New("boom")

				return &fakeBus{devices: []device.TestHandle{dev}}
			},
			err:  true,
			says: "incoming",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			bus := tt.bus()

			got, err := device.OpenOver(context.Background(), bus)

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

			var first *fakeHandle
			if len(bus.devices) > 0 {
				first, _ = bus.devices[0].(*fakeHandle)
			}

			if tt.claims > 0 {
				s.Require().Equal(tt.claims, first.claims)
			}

			if tt.released > 0 {
				s.Require().Equal(tt.released, first.released,
					"the interface is given back")
			}

			if tt.unusedClosed {
				second, _ := bus.devices[1].(*fakeHandle)

				s.Require().False(first.closed)
				s.Require().True(second.closed, "the one nobody used is given back")
			}

			if tt.busClosed {
				s.Require().True(bus.closed, "a bus nobody is using is given back")
			}

			if tt.unclaimed {
				s.Require().Zero(first.claims, "nothing is claimed")
			}

			if tt.allClosed {
				for i, d := range bus.devices {
					h, _ := d.(*fakeHandle)
					s.Require().True(h.closed, "handle %d is given back", i)
				}
			}
		})
	}
}

// TestOpenFindsItsOwnBus covers the one line in this package that reaches
// hardware.
func (s *DiscoverBusPublicTestSuite) TestOpenFindsItsOwnBus() {
	restore := *device.NewBus
	defer func() { *device.NewBus = restore }()

	*device.NewBus = func() device.TestBus {
		return &fakeBus{devices: []device.TestHandle{helix(answers())}}
	}

	got, err := device.Open(context.Background())

	s.Require().NoError(err)
	s.Require().NotNil(got)
}

func TestDiscoverBusTestSuite(t *testing.T) {
	suite.Run(t, new(DiscoverBusPublicTestSuite))
}
