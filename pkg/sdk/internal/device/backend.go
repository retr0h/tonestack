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

package device

import (
	"context"
	"fmt"
	"time"
)

// What a USB backend decides, kept apart from the calls it makes.
//
// A backend file is translation into one operating system's USB API, which no
// test reaches without hardware. Anything holding a decision lives here
// instead, over plain functions a test supplies, so a backend for another
// operating system gets the same behaviour without writing it again.

// matching keeps the entries match accepts and releases the rest.
//
// A USB listing hands back a reference for every device on the bus, and each
// one not kept is one the operating system is still holding for this process.
func matching[T any](
	all []T,
	ids func(T) (vendor, product uint16),
	match func(vendor, product uint16) bool,
	release func(T),
) []T {
	var out []T

	for _, x := range all {
		if v, p := ids(x); match(v, p) {
			out = append(out, x)

			continue
		}

		release(x)
	}

	return out
}

// pickFirst takes the first entry and releases the rest.
func pickFirst[T any](all []T, release func(T)) (T, bool) {
	var zero T

	if len(all) == 0 {
		return zero, false
	}

	for _, extra := range all[1:] {
		release(extra)
	}

	return all[0], true
}

// listed describes every device a listing returned, and gives each back.
//
// Listing is for showing what is attached, so nothing is kept.
func listed[T any](
	all []T,
	err error,
	describe func(T) Descriptor,
	release func(T),
) ([]Descriptor, error) {
	if err != nil {
		return nil, fmt.Errorf("enumerating usb devices: %w", err)
	}

	out := make([]Descriptor, 0, len(all))

	for _, x := range all {
		out = append(out, describe(x))
		release(x)
	}

	return out, nil
}

// found keeps the devices a listing returned that match accepts, as handles.
func found[T any](
	all []T,
	err error,
	ids func(T) (vendor, product uint16),
	match func(vendor, product uint16) bool,
	release func(T),
	wrap func(T) handle,
) ([]handle, error) {
	if err != nil {
		return nil, fmt.Errorf("enumerating usb devices: %w", err)
	}

	var out []handle

	for _, x := range matching(all, ids, match, release) {
		out = append(out, wrap(x))
	}

	return out, nil
}

// claimOne opens the interface a lookup found, and gives back the rest.
//
// An interface that will not open is given back too, and the refusal says
// why, because the next attempt has to start from nothing held.
func claimOne[T any](
	ifaces []T,
	err error,
	number uint8,
	release func(T),
	open func(T) error,
	busy func(error) bool,
) (T, error) {
	var zero T

	if err != nil {
		return zero, fmt.Errorf("finding the editor interface: %w", err)
	}

	intf, ok := pickFirst(ifaces, release)
	if !ok {
		return zero, fmt.Errorf("the device has no interface %d", number)
	}

	if err := open(intf); err != nil {
		release(intf)

		return zero, refused(err, busy(err))
	}

	return intf, nil
}

// piped turns a pipe lookup into the endpoint a session talks over.
func piped[S any](ref uint8, err error, wrap func(uint8) S) (S, error) {
	var zero S

	if err != nil {
		return zero, err
	}

	return wrap(ref), nil
}

// readUntil waits for a read that returns something, or for ctx to end.
//
// A read on a USB pipe takes a timeout rather than a context, and one that
// times out with nothing is the usual case: the device had nothing to say
// yet. So it reads again, one slice at a time, looking at ctx between slices.
// Keeping a read posted like this is one of the rules in docs/protocol.md.
func readUntil(
	ctx context.Context,
	p []byte,
	slice time.Duration,
	read func([]byte, time.Duration) (int, error),
	idle func(error) bool,
) (int, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		n, err := read(p, slice)
		if n == 0 && err != nil && idle(err) {
			continue
		}

		return n, err
	}
}

// refused explains why the editor interface could not be claimed.
//
// Busy means somebody else holds it, and on every Helix that is HX Edit. The
// interface is never seized from it, so saying what to quit is the fix.
func refused(err error, busy bool) error {
	if busy {
		return fmt.Errorf("the editor interface is in use, quit HX Edit: %w", err)
	}

	return fmt.Errorf("claiming the editor interface: %w", err)
}

// located names a device by a location ID, which is how IOKit places a device
// on the bus rather than by bus number and address. The top byte is the bus,
// and the port path below it stands in for the address.
func located(vendor, product uint16, location uint32) Descriptor {
	return Descriptor{
		Vendor:  vendor,
		Product: product,
		Bus:     int(location >> 24),
		Address: int(location & 0x00FFFFFF),
	}
}
