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
	"io"
	"testing"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// Budgets are how long a session waits on each thing, exported so a test
// builds a session with its own rather than writing a package variable.
type Budgets struct {
	Reply     time.Duration
	Commit    time.Duration
	Flash     time.Duration
	Drain     time.Duration
	Close     time.Duration
	Selecting time.Duration
	Poll      time.Duration
	Open      time.Duration
	Pace      time.Duration
	Idle      time.Duration
	Window    time.Duration
	// After is the clock the flash pause, pacing and the select poll wait
	// on. Nil is the real one.
	After func(time.Duration) <-chan time.Time
}

// ShortBudgets are waits a scripted device fits inside, since nothing here is
// talking to hardware.
//
// The ordering is the one a device has: a commit outlasts a reply, and a
// write is waited on for the commit budget. The idle acknowledgement is held
// off, so an answer a test scripted is not acknowledged and thrown away before
// the call it answers; the suites about it shorten it.
func ShortBudgets() Budgets {
	return Budgets{
		Reply:     50 * time.Millisecond,
		Commit:    500 * time.Millisecond,
		Drain:     50 * time.Millisecond,
		Close:     500 * time.Millisecond,
		Selecting: 50 * time.Millisecond,
		Poll:      time.Millisecond,
		Open:      5 * time.Millisecond,
		Pace:      5 * time.Millisecond,
		Idle:      time.Hour,
		Window:    5 * time.Millisecond,
	}
}

// budgets is b as a session holds it.
func (b Budgets) budgets() budgets {
	return budgets{
		reply:     b.Reply,
		commit:    b.Commit,
		flash:     b.Flash,
		drain:     b.Drain,
		close:     b.Close,
		selecting: b.Selecting,
		poll:      b.Poll,
		open:      b.Open,
		pace:      b.Pace,
		idle:      b.Idle,
		window:    b.Window,
		after:     b.After,
	}
}

// NewTestSession builds a session over the given endpoints with the test
// budgets, so the protocol above them can be exercised against a scripted
// device.
//
// Everything a session does apart from finding and claiming hardware happens
// here: framing, sequence numbers, acknowledgements, opening a channel and
// making a call.
func NewTestSession(
	t testing.TB,
	out sender,
	in receiver,
) *session {
	return NewTestSessionWith(t, out, in, ShortBudgets())
}

// NewTestSessionWith is NewTestSession with budgets of the test's own.
//
// A session with somewhere to read from starts its loop at once, as one that
// claimed hardware does, and is closed when the test ends so no loop outlives
// the double it reads. A test that asserts on Close calls it first.
func NewTestSessionWith(
	t testing.TB,
	out sender,
	in receiver,
	b Budgets,
) *session {
	return testSession(t, out, in, b, false)
}

// NewOpenTestSession is NewTestSessionWith with every channel already open
// when the loop starts, as a handshake leaves them. Opened afterwards, a frame
// the loop read first would be dropped as belonging to nobody.
func NewOpenTestSession(
	t testing.TB,
	out sender,
	in receiver,
	b Budgets,
) *session {
	return testSession(t, out, in, b, true)
}

// testSession builds a session for a test, opening its channels first when
// asked to.
func testSession(
	t testing.TB,
	out sender,
	in receiver,
	b Budgets,
	opened bool,
) *session {
	s := newSession(out, in, nil, Model{Name: "HX Stomp"}, b.budgets())

	if opened {
		s.OpenChannels()
	}

	if in != nil {
		s.start()
		// Close is idempotent, and a test that cares what it reports has
		// already asked.
		t.Cleanup(func() { _ = s.Close() })
	}

	return s
}

// Holding records what a session took to reach a device, so that releasing
// it can be tested without one.
func (s *session) Holding(
	held ...func() error,
) {
	for _, release := range held {
		s.holds = append(s.holds, releaseFunc(release))
	}
}

// OnDone records what a session does with the interface it claimed.
func (s *session) OnDone(
	done func(),
) {
	s.done = done
}

// releaseFunc makes a function into something a session can give back.
type releaseFunc func() error

func (f releaseFunc) Close() error { return f() }

// Handshake opens every channel the editor uses.
func (s *session) Handshake(
	ctx context.Context,
) error {
	return s.handshake(ctx)
}

// Drain waits until the device has nothing left to say.
func (s *session) Drain(
	ctx context.Context,
) {
	s.drain(ctx)
}

// Trace sends the wire trace to w, which is how both directions were read off
// a device in the first place.
func (s *session) Trace(
	w io.Writer,
) {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()

	s.trace = w
}

// Received is how many stream bytes the loop has routed to a channel.
func (s *session) Received(
	name string,
) uint32 {
	return s.chans[name].rxBytes.Load()
}

// Buffered is how many bytes a channel holds that nobody has taken.
func (s *session) Buffered(
	name string,
) int {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	return len(s.chans[name].buf)
}

// Transfers is how many reads have brought something.
func (s *session) Transfers() uint64 { return s.progress().transfers }

// Windows is how many reads the loop has finished.
func (s *session) Windows() uint64 { return s.progress().windows }

// IdlePasses is how many rounds the acknowledger has made, so a test waits
// for it to have looked rather than for time to pass.
func (s *session) IdlePasses() uint64 { return s.passes.Load() }

// LoopDone closes once the read loop has returned.
func (s *session) LoopDone() <-chan struct{} { return s.loopDone }

// Dead closes when the read loop ends on its own.
func (s *session) Dead() <-chan struct{} { return s.dead }

// Ended is what ended the read loop, or nil while it runs.
func (s *session) Ended() error { return s.ended() }

// ControlChannel is the channel calls are made on.
const ControlChannel = channelControl

// EventsChannel is the channel the device talks on unasked.
const EventsChannel = channelEvents

// DataChannel is the channel presets are written on.
const DataChannel = channelData

// FrameFor renders a frame the way a device would answer on a channel.
func FrameFor(
	name string,
	msgType uint16,
	payload []byte,
) []byte {
	for _, spec := range channelSpecs {
		if spec.name != name {
			continue
		}

		return wire.EncodeFrame(wire.Frame{
			Flags:      wire.FlagNormal,
			DeviceNode: spec.host,
			HostNode:   spec.device,
			Type:       msgType,
			Payload:    payload,
		})
	}

	return nil
}

// AckFrameFor renders a bare acknowledgement the way a device would send it
// on a channel, carrying the given value.
func AckFrameFor(
	name string,
	ack uint32,
) []byte {
	for _, spec := range channelSpecs {
		if spec.name != name {
			continue
		}

		return wire.EncodeFrame(wire.Frame{
			Flags:      wire.FlagNormal,
			DeviceNode: spec.host,
			HostNode:   spec.device,
			Type:       wire.MsgAck,
			Ack:        ack,
		})
	}

	return nil
}

// ChannelOf names the channel a frame the host sent went out on.
func ChannelOf(
	f wire.Frame,
) string {
	for _, spec := range channelSpecs {
		if f.DeviceNode == spec.device {
			return spec.name
		}
	}

	return ""
}

// Reply renders a device's answer to one call, framed on a channel the way
// the device would send it.
func Reply(
	name string,
	body []byte,
) []byte {
	return FrameFor(name, wire.MsgData, wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromDevice, Service: 2, Body: body,
	}))
}

// OpenChannels opens every channel without a handshake, so a call can be
// tested without scripting one first.
func (s *session) OpenChannels() {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	for _, c := range s.chans {
		c.open = true
	}
}

// MessageKind reads the message type out of a frame a session sent.
func MessageKind(
	frame []byte,
) (uint16, error) {
	f, _, err := wire.DecodeFrame(frame)

	return f.Type, err
}

// ChannelNames is every channel a session opens.
func ChannelNames() []string {
	out := make([]string, 0, len(channelSpecs))
	for _, spec := range channelSpecs {
		out = append(out, spec.name)
	}

	return out
}

// FirstTxn is the transaction number a channel starts at.
const FirstTxn = wire.FirstTxn

// FailAfter makes a device accept a number of writes and refuse the rest, so
// a failure partway through an exchange can be tested.
type FailAfter struct {
	Sender sender
	OK     int
	Err    error
}

func (f *FailAfter) Write(
	p []byte,
) (int, error) {
	if f.OK <= 0 {
		return 0, f.Err
	}

	f.OK--

	return f.Sender.Write(p)
}

// Retry runs something until it works, or until patience runs out.
var Retry = retry

// ClaimAttempts is how many times a busy interface is waited on.
const ClaimAttempts = claimAttempts

// Bus, Handle, Endpoints and Buses are what finding a device runs against,
// exported so a test can supply them.
type (
	TestBus       = bus
	TestHandle    = handle
	TestEndpoints = endpoints
	TestBuses     = buses
)

// OpenOver starts a session over the given bus, with the test budgets.
func OpenOver(
	ctx context.Context,
	b bus,
) (Editor, error) {
	return open(ctx, b, nil, ShortBudgets().budgets())
}

// NewUSBOver is NewUSB taking its buses from source, with the test budgets.
func NewUSBOver(
	trace io.Writer,
	source buses,
) Opener {
	return usbOpener{trace: trace, buses: source, budgets: ShortBudgets().budgets()}
}

// USBBus is the bus NewUSB reaches hardware through. Opening one reads
// nothing; its Devices is what enumerates.
func USBBus() TestBus { return usbBuses{}.Bus() }

// TestSender and TestReceiver are the endpoints a session talks over.
type (
	TestSender   = sender
	TestReceiver = receiver
)

// StreamChunk is how much of a message a device takes per frame.
const StreamChunk = streamChunk

// Write sends a request too large for one frame and waits for the device to
// finish acting on it.
func (s *session) Write(
	ctx context.Context,
	opcode uint64,
	args []wire.Arg,
) error {
	return s.write(ctx, opcode, args)
}

// Exposed to this package's external tests.
//
// Session is the concrete type behind Editor. A caller is handed the
// interface and never names this, so it is not part of what the package
// promises, but the tests that drive a scripted device need the type itself.
type Session = session

var (
	First    = first
	ModelFor = modelFor
)

const VendorID = vendorID

// Matching keeps the entries match accepts and releases the rest.
func Matching[T any](
	all []T,
	ids func(T) (vendor, product uint16),
	match func(vendor, product uint16) bool,
	release func(T),
) []T {
	return matching(all, ids, match, release)
}

// PickFirst takes the first entry and releases the rest.
func PickFirst[T any](
	all []T,
	release func(T),
) (T, bool) {
	return pickFirst(all, release)
}

// ReadUntil waits for a read that returns something, or for ctx to end.
func ReadUntil(
	ctx context.Context,
	p []byte,
	slice time.Duration,
	read func([]byte, time.Duration) (int, error),
	idle func(error) bool,
) (int, error) {
	return readUntil(ctx, p, slice, read, idle)
}

// Refused explains why the editor interface could not be claimed.
func Refused(
	err error,
	busy bool,
) error {
	return refused(err, busy)
}

// Located names a device by a location ID.
func Located(
	vendor, product uint16,
	location uint32,
) Descriptor {
	return located(vendor, product, location)
}

// Listed describes every device a listing returned, and gives each back.
func Listed[T any](
	all []T,
	err error,
	describe func(T) Descriptor,
	release func(T),
) ([]Descriptor, error) {
	return listed(all, err, describe, release)
}

// Found keeps the matching devices and reports how many handles came back.
func Found[T any](
	all []T,
	err error,
	ids func(T) (vendor, product uint16),
	match func(vendor, product uint16) bool,
	release func(T),
) (int, error) {
	hs, err := found(all, err, ids, match, release, func(T) handle { return nil })

	return len(hs), err
}

// ClaimOne opens the interface a lookup found, and gives back the rest.
func ClaimOne[T any](
	ifaces []T,
	err error,
	number uint8,
	release func(T),
	open func(T) error,
	busy func(error) bool,
) (T, error) {
	return claimOne(ifaces, err, number, release, open, busy)
}

// Piped turns a pipe lookup into an endpoint.
func Piped[S any](
	ref uint8,
	err error,
	wrap func(uint8) S,
) (S, error) {
	return piped(ref, err, wrap)
}

// Search looks on the real bus for devices matching nothing, which reads the
// registry through a backend's bus without opening anything.
func Search() (int, error) {
	b := openUSB()
	defer func() { _ = b.Close() }()

	hs, err := b.Devices(func(uint16, uint16) bool { return false })

	return len(hs), err
}
