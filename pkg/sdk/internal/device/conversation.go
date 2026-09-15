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
// device. Finding and claiming hardware lives in usb_darwin.go.
package device

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
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

// Figures arrived at against real hardware. The waits are the defaults of a
// session's budgets, so a test shortens its own rather than a package
// variable.
const (
	openReadWait  = 800 * time.Millisecond
	replyReadWait = 300 * time.Millisecond
	// paceReadWait is the longest a chunk waits for the device to
	// acknowledge the one before it. The pedal acks within one read; this is
	// a device that has stopped taking the message, not one still thinking.
	paceReadWait   = 2 * time.Second
	drainReadWait  = 150 * time.Millisecond
	drainQuietRuns = 3
	claimAttempts  = 7
	claimBackoff   = 50 * time.Millisecond
)

// budgets are how long a session waits on each thing it waits for.
//
// A field of the session rather than package variables, so a test shortens
// its own session's without reaching anybody else's.
type budgets struct {
	// reply is how long a device is given to answer a call.
	reply time.Duration
	// commit is how long a device is given to finish a write.
	//
	// A write answers immediately to say it was accepted and reports finishing
	// later. Treating the first answer as the end races the next write against
	// a commit still running, which a device tolerates about a dozen times
	// before it stops accepting writes.
	commit time.Duration
	// flash is how long a write is given to reach flash before the next one
	// starts.
	flash time.Duration
	// drain bounds one drain. It is bounded because an unbounded one
	// coincided with devices locking up hard enough to need their power
	// pulled.
	drain time.Duration
	// close bounds ending a session. Two drains and the frames between them
	// fit inside it; a device that never goes quiet does not hold Close open
	// past it.
	close time.Duration
	// selecting is how long a switch is given to land, and poll how often the
	// device is asked whether it has.
	selecting time.Duration
	poll      time.Duration
	// open is how long a handshake waits for the device to answer an opening.
	open time.Duration
	// pace is the most a message waits between two chunks for the device to
	// say something.
	pace time.Duration
	// idle is how long a channel nobody is using stays quiet before what
	// arrived on it is acknowledged.
	idle time.Duration
	// window is one read the loop posts. A drain counts quiet ones.
	window time.Duration
	// after is the clock the flash pause, pacing and the select poll wait on.
	// Nil is the real one.
	after func(time.Duration) <-chan time.Time
}

// defaultBudgets are the figures real hardware needs.
func defaultBudgets() budgets {
	return budgets{
		reply:     6 * time.Second,
		commit:    10 * time.Second,
		flash:     750 * time.Millisecond,
		drain:     3 * time.Second,
		close:     10 * time.Second,
		selecting: 10 * time.Second,
		poll:      150 * time.Millisecond,
		open:      openReadWait,
		pace:      paceReadWait,
		idle:      replyReadWait,
		window:    drainReadWait,
	}
}

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

// channelEvents carries what the device says unasked. Nothing here reads it,
// so its bytes are counted and acknowledged and never kept.
const channelEvents = "events"

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
	{channelEvents, 0x1002, 0x03f0, []uint16{4}},
	{channelData, 0x1080, 0x03ed, []uint16{6}},
}

// helloTail is the four bytes a channel opening carries, and helloAck sits in
// its acknowledgement slot. Neither is understood. Both are sent because HX
// Edit sends them and the device answers.
var helloTail = []byte{0x00, 0x10, 0x00, 0x00}

const helloAck uint32 = 0x21000100

// firstSeq is where a channel's counter goes after its opening frame.
//
// Two, not one: HX Edit's counter jumps from zero straight to two, and the
// device stops answering a client that sends one.
const firstSeq = 2

// channel is one conversation with the device.
//
// seq and txn change only under the session's send lock, and buf, open,
// lastRx, lastAck and ackSeen only under its receive lock. rxBytes, ackSent
// and rxAcks are read from both sides, so they are atomic.
type channel struct {
	name   string
	device uint16
	host   uint16
	seq    uint16
	txn    uint64
	// rxBytes is every stream byte routed here, and ackSent what the last
	// acknowledgement this host sent on the channel said it had.
	rxBytes atomic.Uint32
	ackSent atomic.Uint32
	// rxAcks counts the device's own acknowledgements on this channel that
	// carried a value different from the one before, which is what a chunk's
	// pace waits to pass. lastAck is that value and ackSeen whether one has
	// arrived yet; the device's ack base has no relation to wire.AckBase, so
	// only a change is ever compared, never a computed byte count.
	rxAcks  atomic.Uint64
	lastAck uint32
	ackSeen bool
	buf     []byte
	// open is whether the handshake has reached this channel. A frame on a
	// channel nobody opened is not anybody's business.
	open bool
	// lastRx is when bytes last arrived.
	lastRx time.Time
	// arrived is signalled whenever bytes are routed here. One slot, so
	// routing never waits and a waiter that was not yet waiting still wakes.
	arrived chan struct{}
}

// owed reports bytes that arrived since this host last acknowledged the
// channel.
func (c *channel) owed() bool { return c.rxBytes.Load() != c.ackSent.Load() }

// acked is how many of the device's own acknowledgements have moved on this
// channel.
func (c *channel) acked() uint64 { return c.rxAcks.Load() }

// session is an open conversation with a device.
//
// One operation at a time. The protocol is a sequence of exchanges with
// per-channel counters, and two callers interleaving exchanges would read
// each other's answers, so whoever holds a session runs one operation after
// another. Inside it, one goroutine keeps a read posted and routes what
// arrives, and another acknowledges what arrives between operations.
type session struct {
	// holds is what the session took to reach the device, released in the
	// order it was taken. Kept as an interface so that a session is a
	// conversation rather than a piece of hardware: everything below this is
	// framing and counters, and none of it needs a bus.
	holds []releaser
	done  func()
	out   sender
	in    receiver
	// chans is every channel, made with the session and never written after,
	// so the loop ranges over it without a lock.
	chans   map[string]*channel
	model   Model
	budgets budgets

	// traceMu guards trace, which the loop and the sender both write. A nil
	// trace receives nothing.
	traceMu sync.Mutex
	trace   io.Writer

	// sendMu covers the sequence counters, the transaction counters and the
	// write itself, so frames leave in the order their numbers say.
	sendMu sync.Mutex

	// rxMu guards every channel's buffer and flags, and everything below it
	// up to stop.
	rxMu sync.Mutex
	// windows counts reads the loop finished, transfers those that brought
	// something, and quietRun how many in a row brought nothing.
	windows   uint64
	transfers uint64
	quietRun  int
	// changed is closed and replaced whenever a read finishes, so anybody
	// waiting on the device selects on it.
	changed chan struct{}
	// closing keeps the idle acknowledgement out while Close is talking.
	closing bool
	// inflight counts exchanges, writes and channel openings under way. While
	// any is, the idle acknowledgement sends nothing on any channel.
	inflight int
	// passes counts the acknowledger's rounds, so a test can wait for it to
	// have looked rather than for time to pass.
	passes atomic.Uint64

	// stop ends the loop and the acknowledger. loopDone and ackDone close
	// once each has returned. Nil stop means neither was started.
	stop     context.CancelFunc
	loopDone chan struct{}
	ackDone  chan struct{}
	// poke tells the acknowledger a read finished.
	poke chan struct{}
	// dead closes when the loop ends on its own, and endErr says why.
	dead    chan struct{}
	endOnce sync.Once
	endErr  error

	closeOnce sync.Once
	closeErr  error
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

// newSession builds a session with every channel made and none of them open.
// Nothing is read until start.
func newSession(
	out sender,
	in receiver,
	trace io.Writer,
	model Model,
	b budgets,
) *session {
	s := &session{
		out:     out,
		in:      in,
		trace:   trace,
		model:   model,
		budgets: b,
		chans:   map[string]*channel{},
		changed: make(chan struct{}),
		poke:    make(chan struct{}, 1),
		dead:    make(chan struct{}),
	}

	for _, spec := range channelSpecs {
		s.chans[spec.name] = &channel{
			name:    spec.name,
			device:  spec.device,
			host:    spec.host,
			txn:     wire.FirstTxn,
			arrived: make(chan struct{}, 1),
		}
	}

	return s
}

// Model returns what the device is.
func (s *session) Model() Model { return s.model }

// tracef writes one line of the wire trace, when there is one.
func (s *session) tracef(
	format string,
	args ...any,
) {
	s.traceMu.Lock()
	defer s.traceMu.Unlock()

	if s.trace != nil {
		fmt.Fprintf(s.trace, format, args...)
	}
}

// Close ends the session.
//
// Whatever the device sent is drained and acknowledged first. Dropping the
// interface with bytes unacknowledged carries a debt into later sessions,
// until an otherwise innocent write stops the device.
//
// Idempotent. Every call gives back what the session took and returns the
// error that ended the read loop, if one did. A session whose loop ended still
// says goodbye, though nothing reads what the device answers.
func (s *session) Close() error {
	s.closeOnce.Do(func() { s.closeErr = s.close() })

	return s.closeErr
}

// close is Close, once.
func (s *session) close() error {
	if s.stop != nil {
		// Attempted even when the bus has failed, as a session did before
		// there was a loop: a device never told the editor is gone keeps its
		// front panel stale. Its drains end at once, since nothing reads.
		s.farewell()

		// The loop reads until here, and only then is the interface let go.
		// ReadContext looks at its context every slice, so this is short.
		s.stop()
		<-s.loopDone
		<-s.ackDone
	}

	if s.done != nil {
		s.done()
	}

	// In the order they were taken: the interface first, then the device,
	// then the library's own context.
	for i, held := range s.holds {
		// Every hold must be released even when an earlier one refuses.
		if err := held.Close(); err != nil {
			s.tracef("ERR release %d on close: %v\n", i, err)
		}
	}

	return s.ended()
}

// farewell tells the device the session is over, while the loop is still
// reading what it answers.
func (s *session) farewell() {
	s.rxMu.Lock()
	s.closing = true
	s.rxMu.Unlock()

	// Bounded as a whole. Each drain is bounded on its own, but a device that
	// never goes quiet would hold Close for both of them.
	ctx, cancel := context.WithTimeout(context.Background(), s.budgets.close)
	defer cancel()

	s.drain(ctx)

	// In the order they were opened, rather than whichever way a map ranges
	// today. A device is told about a session ending in a sequence, and a
	// sequence that differs between runs is one nobody can compare against a
	// capture.
	for _, c := range s.opened() {
		// Close has nobody to tell, and one failed send must not stop the
		// rest of the shutdown. The wire trace is the one place it shows.
		if err := s.farewellFrame(c, wire.MsgAck); err != nil {
			s.tracef("ERR %-8s ack on close: %v\n", c.name, err)
		}
	}

	// The message that opens a channel closes one: it is a session boundary
	// and appears at both ends of the conversation. A device left without it
	// goes on believing an editor is attached, and its front panel stops
	// refreshing footswitches as somebody browses presets on the pedal
	// itself.
	for _, c := range s.opened() {
		// For the same reason: every other channel still needs closing.
		if err := s.farewellFrame(c, wire.MsgHello); err != nil {
			s.tracef("ERR %-8s hello on close: %v\n", c.name, err)
		}
	}

	// The device answers each one. Reading them is what makes the next
	// session's handshake the first thing it sees rather than the last thing
	// this one left.
	s.drain(ctx)
}

// farewellFrame sends one closing frame.
func (s *session) farewellFrame(
	c *channel,
	msgType uint16,
) error {
	return s.send(c, msgType, nil)
}

// opened is every channel the handshake reached, in the order they open.
func (s *session) opened() []*channel {
	s.rxMu.Lock()
	defer s.rxMu.Unlock()

	var out []*channel

	for _, spec := range channelSpecs {
		if c := s.chans[spec.name]; c.open {
			out = append(out, c)
		}
	}

	return out
}

// retry runs something until it works, or until patience runs out.
//
// Cleanup after a previous session races the next claim, so an interface that
// is busy is worth waiting on rather than reporting. Separate from the call
// itself because the policy — how many times, how long between — is the part
// worth being sure about, and the call is the part that needs hardware.
func retry(
	attempt func() error,
) error {
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
