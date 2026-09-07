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

// The three usb_*.go files are the only ones in this package that touch
// libusb, and the only ones a test cannot reach. Everything else — framing,
// sequence numbers, acknowledgements, opening a channel, making a call — takes
// its endpoints as interfaces and runs against a scripted device.
package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/gousb"
)

// Open starts a session with the first attached device.
//
// HX Edit must be quit first: it claims the editor interface exclusively.
//
// Returns the interface rather than the type behind it, so that everything
// above this package can be given a session instead of finding one — which is
// what lets reading a device be tested without one attached.
func Open(ctx context.Context) (Editor, error) {
	uctx := gousb.NewContext()

	dev, model, err := findDevice(uctx)
	if err != nil {
		_ = uctx.Close()

		return nil, err
	}

	s := &Session{holds: []releaser{dev, uctx}, model: model, chans: map[string]*channel{}}

	if err := s.claim(dev); err != nil {
		s.Close()

		return nil, err
	}

	if err := s.handshake(ctx); err != nil {
		s.Close()

		return nil, err
	}

	return s, nil
}

// findDevice opens the first device this package recognises.
func findDevice(uctx *gousb.Context) (*gousb.Device, Model, error) {
	var (
		found *gousb.Device
		model Model
	)

	devs, err := uctx.OpenDevices(func(d *gousb.DeviceDesc) bool {
		_, ok := ModelFor(uint16(d.Product))

		return ok
	})
	if err != nil && len(devs) == 0 {
		return nil, Model{}, fmt.Errorf("looking for a device: %w", err)
	}

	for i, d := range devs {
		if i > 0 {
			_ = d.Close()

			continue
		}

		found = d
		model, _ = ModelFor(uint16(d.Desc.Product))
	}

	if found == nil {
		return nil, Model{}, ErrNoDevice
	}

	return found, model, nil
}

// claim takes the editor interface.
//
// Claimed, released, and claimed again, which is what HX Edit does. It looks
// like startup noise until reconnecting without it fails on roughly every
// other attempt: the device carries channel state across connections, and the
// release is what clears it.
func (s *Session) claim(dev *gousb.Device) error {
	_, release, err := claimOnce(dev)
	if err != nil {
		return err
	}

	release()

	intf, release, err := claimOnce(dev)
	if err != nil {
		return err
	}

	s.done = release

	if s.out, err = intf.OutEndpoint(endpointOut); err != nil {
		return fmt.Errorf("opening the outgoing endpoint: %w", err)
	}

	if s.in, err = intf.InEndpoint(endpointIn); err != nil {
		return fmt.Errorf("opening the incoming endpoint: %w", err)
	}

	return nil
}

// claimOnce takes the interface, retrying while it is busy.
//
// Cleanup after a previous session races the next claim, so a busy interface
// is worth waiting on rather than reporting.
func claimOnce(dev *gousb.Device) (*gousb.Interface, func(), error) {
	var last error

	for range claimAttempts {
		intf, release, err := dev.DefaultInterface()
		if err == nil {
			return intf, release, nil
		}

		last = err

		time.Sleep(claimBackoff)
	}

	return nil, nil, fmt.Errorf(
		"claiming the editor interface (is HX Edit running?): %w", last)
}
