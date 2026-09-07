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

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// NewTestSession builds a session over the given endpoints, so the protocol
// above them can be exercised against a scripted device.
//
// Everything a session does apart from finding and claiming hardware happens
// here: framing, sequence numbers, acknowledgements, opening a channel and
// making a call.
func NewTestSession(out sender, in receiver) *Session {
	return &Session{
		out:   out,
		in:    in,
		chans: map[string]*channel{},
		model: Model{Name: "HX Stomp"},
	}
}

// Holding records what a session took to reach a device, so that releasing
// it can be tested without one.
func (s *Session) Holding(held ...func() error) {
	for _, release := range held {
		s.holds = append(s.holds, releaseFunc(release))
	}
}

// OnDone records what a session does with the interface it claimed.
func (s *Session) OnDone(done func()) { s.done = done }

// releaseFunc makes a function into something a session can give back.
type releaseFunc func() error

func (f releaseFunc) Close() error { return f() }

// Handshake opens every channel the editor uses.
func (s *Session) Handshake(ctx context.Context) error { return s.handshake(ctx) }

// Drain reads until the device has nothing left to say.
func (s *Session) Drain(ctx context.Context) { s.drain(ctx) }

// Receive reads one transfer and routes every frame in it.
func (s *Session) Receive(ctx context.Context) bool {
	return s.receive(ctx, openReadWait)
}

// SetDebug turns the wire trace on, which is how both directions were read
// off a device in the first place.
func SetDebug(on bool) func() {
	was := debug
	debug = on

	return func() { debug = was }
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
func (s *Session) OpenChannels() {
	for _, spec := range channelSpecs {
		s.chans[spec.name] = &channel{
			name: spec.name, device: spec.device, host: spec.host,
			txn: wire.FirstTxn,
		}
	}
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
func OpenOver(ctx context.Context, b bus) (Editor, error) { return open(ctx, b) }

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

// CommitBudget is how long a device is given to finish a write, exported so a
// test need not wait the whole of it.
var CommitBudget = &commitBudget

// Write sends a request too large for one frame and waits for the device to
// finish acting on it.
func (s *Session) Write(ctx context.Context, opcode uint64, args []wire.Arg) error {
	return s.write(ctx, opcode, args)
}
