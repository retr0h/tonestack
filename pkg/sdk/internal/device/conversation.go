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

// This file is the conversation: what a session says and how it counts. It
// takes its endpoints as interfaces, so all of it runs against a scripted
// device. Finding and claiming hardware lives in usb_open.go.
package device

import (
	"context"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
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
	drainReadWait  = 150 * time.Millisecond
	drainBudget    = 3 * time.Second
	drainQuietRuns = 3
	claimAttempts  = 7
	claimBackoff   = 50 * time.Millisecond
)

// replyBudget is how long a device is given to answer. A variable rather than
// a constant so a test can shorten it; nothing else writes to it.
var replyBudget = 6 * time.Second

// Opcodes this package uses.
const (
	opListPresets = 1
	// opReadPreset reads a slot without loading it: the device hands back
	// the document and goes on playing whatever it was.
	opReadPreset = 4
)

// Argument keys for those opcodes.
const (
	argSetlist  = 107
	argListKind = 101
	argSlot     = 108
)

// listKind is what HX Edit always sends alongside a setlist. Its meaning is
// not known; the device is shown it because the device has been shown it.
const listKind = 2

// The three channels, and the services opened on each.
//
// The control channel is opened twice: once for service 5, which is then
// closed, and again from scratch for service 2. Requests sent to service 5
// time out silently, which is easy to mistake for a flaky device.
// channelControl carries session control and the setlist list.
const channelControl = "control"

// channelData carries presets and global settings.
//
// Reading a preset works on either, which is how every read here was written
// against the control channel and passed. Writing one does not: a device
// answers a write on the control channel with error -3 and changes nothing.
// tonepush sends every preset operation here.
const channelData = "data"

var channelSpecs = []struct {
	name     string
	device   uint16
	host     uint16
	services []uint16
}{
	{channelControl, 0x1001, 0x03ef, []uint16{5, 2}},
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

// session is an open conversation with a device.
//
// Not safe for concurrent use. The protocol is a sequence of exchanges with
// per-channel counters, and two callers sharing one would desynchronise them.
type session struct {
	// holds is what the session took to reach the device, released in the
	// order it was taken. Kept as an interface so that a session is a
	// conversation rather than a piece of hardware: everything below this is
	// framing and counters, and none of it needs a bus.
	holds []releaser
	done  func()
	out   sender
	in    receiver
	chans map[string]*channel
	model Model
}

// sender is the outgoing endpoint: everything this writes goes to a device.
//
// The two endpoints are interfaces so that the protocol above them — framing,
// sequence numbers, acknowledgements, opening a channel — can be exercised
// against a scripted device instead of a real one. Only finding and claiming
// hardware needs the real thing.
type sender interface {
	Write(p []byte) (int, error)
}

// releaser is something taken to reach a device and given back afterwards.
type releaser interface {
	Close() error
}

// receiver is the incoming endpoint.
type receiver interface {
	ReadContext(ctx context.Context, p []byte) (int, error)
}

// Model returns what the device is.
func (s *session) Model() Model { return s.model }

// Close ends the session.
//
// Whatever the device sent is drained and acknowledged first. Dropping the
// interface with bytes unacknowledged carries a debt into later sessions,
// until an otherwise innocent write stops the device.
func (s *session) Close() {
	if s.in != nil {
		ctx := context.Background()

		s.drain(ctx)

		// In the order they were opened, rather than whichever way a map
		// ranges today. A device is told about a session ending in a
		// sequence, and a sequence that differs between runs is one nobody
		// can compare against a capture.
		for _, spec := range channelSpecs {
			if c, ok := s.chans[spec.name]; ok {
				_ = s.send(c, wire.MsgAck, nil)
			}
		}

		// The message that opens a channel closes one: it is a session
		// boundary and appears at both ends of the conversation. A device
		// left without it goes on believing an editor is attached, and its
		// front panel stops refreshing footswitches as somebody browses
		// presets on the pedal itself.
		for _, spec := range channelSpecs {
			if c, ok := s.chans[spec.name]; ok {
				_ = s.closeChannel(c)
			}
		}

		// The device answers each one. Reading them is what makes the next
		// session's handshake the first thing it sees rather than the last
		// thing this one left.
		s.drain(ctx)
	}

	if s.done != nil {
		s.done()
	}

	// In the order they were taken: the interface first, then the device,
	// then the library's own context.
	for _, held := range s.holds {
		_ = held.Close()
	}
}

// retry runs something until it works, or until patience runs out.
//
// Cleanup after a previous session races the next claim, so an interface that
// is busy is worth waiting on rather than reporting. Separate from the call
// itself because the policy — how many times, how long between — is the part
// worth being sure about, and the call is the part that needs hardware.
func retry(attempt func() error) error {
	var last error

	for i := range claimAttempts {
		if last = attempt(); last == nil {
			return nil
		}

		// Not after the last one: nobody is waiting for anything then.
		if i < claimAttempts-1 {
			time.Sleep(claimBackoff)
		}
	}

	return last
}
