//go:build cgo

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
	"time"

	"github.com/google/gousb"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// The editor endpoint. Interface 0 is vendor-specific and carries the editor
// protocol; the MIDI interface is a different one and cannot do any of this.
const (
	endpointOut = 0x01
	endpointIn  = 0x81
	readBuffer  = 1024
)

// Timeouts, every one a figure arrived at against real hardware. The drain
// budget is bounded because an unbounded one coincided with devices locking
// up hard enough to need their power pulled.
const (
	openReadWait   = 800 * time.Millisecond
	replyReadWait  = 300 * time.Millisecond
	replyBudget    = 6 * time.Second
	drainReadWait  = 150 * time.Millisecond
	drainBudget    = 3 * time.Second
	drainQuietRuns = 3
	claimAttempts  = 7
	claimBackoff   = 50 * time.Millisecond
)

// Opcodes this package uses.
const (
	opListPresets = 1
)

// Argument keys for those opcodes.
const (
	argSetlist  = 107
	argListKind = 101
)

// listKind is what HX Edit always sends alongside a setlist. Its meaning is
// not known; the device is shown it because the device has been shown it.
const listKind = 2

// The three channels, and the services opened on each.
//
// The control channel is opened twice: once for service 5, which is then
// closed, and again from scratch for service 2. Requests sent to service 5
// time out silently, which is easy to mistake for a flaky device.
var channelSpecs = []struct {
	name     string
	device   uint16
	host     uint16
	services []uint16
}{
	{"control", 0x1001, 0x03ef, []uint16{5, 2}},
	{"events", 0x1002, 0x03f0, []uint16{4}},
	{"data", 0x1080, 0x03ed, []uint16{6}},
}

// helloTail is the four bytes a channel opening carries, and helloAck sits in
// its acknowledgement slot. Neither is understood. Both are sent because HX
// Edit sends them and the device answers.
var helloTail = []byte{0x00, 0x10, 0x00, 0x00}

const helloAck uint32 = 0x21000100

// firstSeq is where a channel's counter goes after its opening frame.
//
// One, not two: HX Edit's counter jumps from zero straight to two, and the
// device stops answering a client that sends one.
const firstSeq = 2

// channel is one conversation with the device.
type channel struct {
	name    string
	device  uint16
	host    uint16
	seq     uint16
	rxBytes uint32
	txn     uint64
	buf     []byte
}

// ack is what this channel has consumed, in the form the device expects.
// Not a bare count: the device ignores a client that sends one.
func (c *channel) ack() uint32 { return wire.AckBase + c.rxBytes }

// Session is an open conversation with a device.
//
// Not safe for concurrent use. The protocol is a sequence of exchanges with
// per-channel counters, and two callers sharing one would desynchronise them.
type Session struct {
	uctx  *gousb.Context
	dev   *gousb.Device
	done  func()
	out   *gousb.OutEndpoint
	in    *gousb.InEndpoint
	chans map[string]*channel
	model Model
}

// Model returns what the device is.
func (s *Session) Model() Model { return s.model }

// Open starts a session with the first attached device.
//
// HX Edit must be quit first: it claims the editor interface exclusively.
func Open(ctx context.Context) (*Session, error) {
	uctx := gousb.NewContext()

	dev, model, err := findDevice(uctx)
	if err != nil {
		_ = uctx.Close()

		return nil, err
	}

	s := &Session{uctx: uctx, dev: dev, model: model, chans: map[string]*channel{}}

	if err := s.claim(); err != nil {
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
func (s *Session) claim() error {
	_, release, err := s.claimOnce()
	if err != nil {
		return err
	}

	release()

	intf, release, err := s.claimOnce()
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
func (s *Session) claimOnce() (*gousb.Interface, func(), error) {
	var last error

	for range claimAttempts {
		intf, release, err := s.dev.DefaultInterface()
		if err == nil {
			return intf, release, nil
		}

		last = err

		time.Sleep(claimBackoff)
	}

	return nil, nil, fmt.Errorf(
		"claiming the editor interface (is HX Edit running?): %w", last)
}

// Close ends the session.
//
// Whatever the device sent is drained and acknowledged first. Dropping the
// interface with bytes unacknowledged carries a debt into later sessions,
// until an otherwise innocent write stops the device.
func (s *Session) Close() {
	if s.in != nil {
		_ = s.drain(context.Background())

		for _, c := range s.chans {
			_ = s.send(c, wire.MsgAck, nil)
		}
	}

	if s.done != nil {
		s.done()
	}

	if s.dev != nil {
		_ = s.dev.Close()
	}

	if s.uctx != nil {
		_ = s.uctx.Close()
	}
}
