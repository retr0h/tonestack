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

package sdk

import (
	"context"
	"fmt"
)

// bus is a USB bus this package can look at.
//
// Declared here rather than taken from the library, so that finding and
// claiming a device is logic over an interface instead of a call into C. What
// remains in usb.go is one expression per method, forwarding.
type bus interface {
	// Devices returns every device the matcher accepts, opened.
	Devices(match func(vendor, product uint16) bool) ([]handle, error)
	// Close releases the library's own context.
	Close() error
}

// handle is one open device.
type handle interface {
	// Descriptor is what the device says it is, without opening anything.
	Descriptor() Descriptor
	// Claim takes the editor interface, and returns how to give it back.
	Claim() (endpoints, func(), error)
	// Close releases the device.
	Close() error
}

// endpoints is the pair a session talks over.
type endpoints interface {
	// Out is where requests go.
	Out() (sender, error)
	// In is where answers come from.
	In() (receiver, error)
}

// newBus is how a bus is obtained, so a test can stand in for it.
//
// The only line in this package that reaches hardware; everything below takes
// what it was given.
var newBus = openUSB

// Open starts a session with the first attached device.
//
// HX Edit must be quit first: it claims the editor interface exclusively.
//
// Returns the interface rather than the type behind it, so that everything
// above this package can be given a session instead of finding one — which is
// what lets reading a device be tested without one attached.
func Open(ctx context.Context) (Editor, error) { return open(ctx, newBus()) }

// open starts a session over the given bus.
//
// The bus is closed on failure and handed to the session on success, because
// a session holds it open for as long as it is talking.
func open(ctx context.Context, b bus) (Editor, error) {
	dev, model, err := findDevice(b)
	if err != nil {
		_ = b.Close()

		return nil, err
	}

	s := &Session{
		holds: []releaser{dev, b},
		model: model,
		chans: map[string]*channel{},
	}

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
//
// More than one is possible and only the first is used. The rest are closed
// rather than left open, because a device held by a process that is not using
// it is a device nothing else can claim.
func findDevice(b bus) (handle, Model, error) {
	devs, err := b.Devices(func(_, product uint16) bool {
		_, ok := ModelFor(product)

		return ok
	})
	if err != nil && len(devs) == 0 {
		return nil, Model{}, fmt.Errorf("looking for a device: %w", err)
	}

	var found handle

	for i, d := range devs {
		if i > 0 {
			_ = d.Close()

			continue
		}

		found = d
	}

	if found == nil {
		return nil, Model{}, ErrNoDevice
	}

	model, _ := ModelFor(found.Descriptor().Product)

	return found, model, nil
}

// claim takes the editor interface.
//
// Claimed, released, and claimed again, which is what HX Edit does. It looks
// like startup noise until reconnecting without it fails on roughly every
// other attempt: the device carries channel state across connections, and the
// release is what clears it.
func (s *Session) claim(dev handle) error {
	_, release, err := claimOnce(dev)
	if err != nil {
		return err
	}

	release()

	ends, release, err := claimOnce(dev)
	if err != nil {
		return err
	}

	s.done = release

	if s.out, err = ends.Out(); err != nil {
		return fmt.Errorf("opening the outgoing endpoint: %w", err)
	}

	if s.in, err = ends.In(); err != nil {
		return fmt.Errorf("opening the incoming endpoint: %w", err)
	}

	return nil
}

// claimOnce takes the interface, waiting while it is busy.
func claimOnce(dev handle) (endpoints, func(), error) {
	var (
		ends    endpoints
		release func()
	)

	err := retry(func() error {
		var err error
		ends, release, err = dev.Claim()

		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf(
			"claiming the editor interface (is HX Edit running?): %w", err)
	}

	return ends, release, nil
}
