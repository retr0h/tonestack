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
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// NewTestSession builds a session over the given endpoints, so the protocol
// above them can be exercised against a scripted device.
//
// Everything a session does apart from finding and claiming hardware happens
// here: framing, sequence numbers, acknowledgements, opening a channel and
// making a call.
func NewTestSession(out sender, in receiver) *session {
	return &session{
		out:   out,
		in:    in,
		chans: map[string]*channel{},
		model: Model{Name: "HX Stomp"},
	}
}

// Holding records what a session took to reach a device, so that releasing
// it can be tested without one.
func (s *session) Holding(held ...func() error) {
	for _, release := range held {
		s.holds = append(s.holds, releaseFunc(release))
	}
}

// OnDone records what a session does with the interface it claimed.
func (s *session) OnDone(done func()) { s.done = done }

// releaseFunc makes a function into something a session can give back.
type releaseFunc func() error

func (f releaseFunc) Close() error { return f() }

// Handshake opens every channel the editor uses.
func (s *session) Handshake(ctx context.Context) error { return s.handshake(ctx) }

// Drain reads until the device has nothing left to say.
func (s *session) Drain(ctx context.Context) { s.drain(ctx) }

// Receive reads one transfer and routes every frame in it.
func (s *session) Receive(
	ctx context.Context,
) (bool, error) {
	return s.receive(ctx, openReadWait)
}

// Trace sends the wire trace to w, which is how both directions were read off
// a device in the first place.
func (s *session) Trace(
	w io.Writer,
) {
	s.trace = w
}

// ControlChannel is the channel calls are made on.
const ControlChannel = "control"

// DataChannel is the channel presets are written on.
const DataChannel = channelData

// FrameFor renders a frame the way a device would answer on a channel.
func FrameFor(name string, msgType uint16, payload []byte) []byte {
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

// Reply renders a device's answer to one call, framed on a channel the way
// the device would send it.
func Reply(name string, body []byte) []byte {
	return FrameFor(name, wire.MsgData, wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromDevice, Service: 2, Body: body,
	}))
}

// OpenChannels puts a channel in place without a handshake, so a call can be
// tested without scripting one first.
func (s *session) OpenChannels() {
	for _, spec := range channelSpecs {
		s.chans[spec.name] = &channel{
			name: spec.name, device: spec.device, host: spec.host,
			txn: wire.FirstTxn,
		}
	}
}

// MessageKind reads the message type out of a frame a session sent.
func MessageKind(frame []byte) (uint16, error) {
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

func (f *FailAfter) Write(p []byte) (int, error) {
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

// Bus, Handle and Endpoints are what finding a device runs against, exported
// so a test can supply them.
type (
	TestBus       = bus
	TestHandle    = handle
	TestEndpoints = endpoints
)

// NewBus is how a bus is obtained, exported so a test can stand in for the
// only line in this package that reaches hardware.
var NewBus = &newBus

// OpenOver starts a session over the given bus.
func OpenOver(ctx context.Context, b bus) (Editor, error) { return open(ctx, b, nil) }

// TestSender and TestReceiver are the endpoints a session talks over.
type (
	TestSender   = sender
	TestReceiver = receiver
)

// StreamChunk is how much of a message a device takes per frame.
const StreamChunk = streamChunk

// FlashBudget is how long a write is given to reach flash, exported so a test
// does not spend it.
var FlashBudget = &flashBudget

// SelectPoll and SelectBudget pace the wait for a switch to land, exported so
// a test does not spend it.
var (
	SelectPoll   = &selectPoll
	SelectBudget = &selectBudget
)

// CommitBudget is how long a device is given to finish a write, exported so a
// test need not wait the whole of it.
var CommitBudget = &commitBudget

// ReplyBudget is how long a device is given to answer a call, exported so a
// test can reach the silence without waiting out the whole of it.
var ReplyBudget = &replyBudget

// DrainBudget bounds how long a drain reads, exported so a test does not
// spend it.
var DrainBudget = &drainBudget

// CloseBudget bounds how long ending a session takes, exported so a test can
// reach it without waiting out the whole of it.
var CloseBudget = &closeBudget

// Write sends a request too large for one frame and waits for the device to
// finish acting on it.
func (s *session) Write(ctx context.Context, opcode uint64, args []wire.Arg) error {
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
func PickFirst[T any](all []T, release func(T)) (T, bool) { return pickFirst(all, release) }

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
func Refused(err error, busy bool) error { return refused(err, busy) }

// Located names a device by a location ID.
func Located(vendor, product uint16, location uint32) Descriptor {
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
func Piped[S any](ref uint8, err error, wrap func(uint8) S) (S, error) {
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
