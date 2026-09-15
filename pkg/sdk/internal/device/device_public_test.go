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
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// ackBase is where the scripted device's acknowledgements start counting
// from. Any base works; this one matches a hardware trace, and the point of
// using one that differs from wire.AckBase is that nothing here should ever
// compare an ack to a computed byte count.
const ackBase uint32 = 0x0fd8

// The scripted device every suite in this package talks to. It answers what
// it was told to answer and records what it was sent, so the protocol can be
// exercised in full with no hardware attached.
//
// The sender and receiver it presents are generated mocks (device.Mocksender
// and device.Mockreceiver), with AnyTimes expectations: a session's loop reads
// at moments no test can establish. deviceDouble only groups them with the
// closures that give them their behaviour, the way CONTRIBUTING's "Test
// doubles" section asks of a type standing in for a generated one.

// deviceDouble is a sender and receiver mock scripted to answer whatever is
// written to it.
//
// Safe for the loop and the test to use at once: every field below mu is read
// and written under it.
type deviceDouble struct {
	// out and in are what a session talks to. NewTestSession takes them
	// directly.
	out *device.Mocksender
	in  *device.Mockreceiver

	mu sync.Mutex
	// replies go out one for each frame the session writes, the way a device
	// answers what it is asked rather than before.
	replies [][]byte
	// ready is what the next reads hand back.
	ready [][]byte
	// sent is everything the session wrote.
	sent [][]byte
	// writeErr fails every write.
	writeErr error
	// readErr fails every read with nothing ready, once failAfter frames have
	// been written.
	readErr   error
	failAfter int
	// partial comes back beside every frame a read hands over: bytes that
	// arrived with a timeout.
	partial error
	// reads is how many times the session read.
	reads int
	// onWrite runs after every write the device took, so a test can act
	// partway through a message.
	onWrite func()
	// ackValue is the last acknowledgement value handed out on a channel, and
	// acked how many of its data frames have earned one, so it can advance by
	// each payload's length and stop counting once a limit is scripted.
	ackValue map[string]uint32
	acked    map[string]int
	// ackLimit caps how many of a channel's data frames earn an automatic
	// acknowledgement; a channel absent from the map earns one for every
	// frame. ackDivert is what the first frame past the limit gets instead of
	// silence, when one is scripted.
	ackLimit  map[string]int
	ackDivert map[string][]byte
	// autoAcked is what the device answers a data frame with automatically,
	// served after ready. Never counted by pending: it is not a scripted
	// reply, and a test that waits for scripted replies to drain must not see
	// pacing as one still outstanding.
	autoAcked [][]byte
	// noisy is handed back on every read with nothing ready: a device that
	// never goes quiet.
	noisy []byte
	// holdFor keeps what is ready back until that long after the first read:
	// a device that answers, but late.
	holdFor time.Duration
	first   time.Time
	// wake tells a read that is waiting that something is ready.
	wake chan struct{}
}

// newDeviceDouble wires a sender and a receiver mock to a device's behaviour.
// Fields are set before a session is built over it.
func newDeviceDouble(
	ctrl *gomock.Controller,
) *deviceDouble {
	d := &deviceDouble{wake: make(chan struct{}, 1)}

	d.out = device.NewMocksender(ctrl)
	d.out.EXPECT().Write(gomock.Any()).DoAndReturn(d.write).AnyTimes()

	d.in = device.NewMockreceiver(ctrl)
	d.in.EXPECT().ReadContext(gomock.Any(), gomock.Any()).DoAndReturn(d.read).AnyTimes()

	return d
}

// write records what the session sent, fails it when writeErr is set, paces
// it the way a device does, and lets the next reply go.
func (d *deviceDouble) write(
	p []byte,
) (int, error) {
	d.mu.Lock()

	if d.writeErr != nil {
		err := d.writeErr
		d.mu.Unlock()

		return 0, err
	}

	d.sent = append(d.sent, append([]byte(nil), p...))
	d.autoAck(p)

	if len(d.replies) > 0 {
		d.ready = append(d.ready, d.replies[0])
		d.replies = d.replies[1:]
	}

	hook := d.onWrite
	d.mu.Unlock()

	d.signal()

	if hook != nil {
		hook()
	}

	return len(p), nil
}

// autoAck makes ready the device's own answer to a chunk of a write the
// session sent: an acknowledgement on the data channel whose value advances
// by the payload's length, the way a real device paces the sender. A channel
// scripted with stopAckingAfter earns that many and then goes quiet, saying
// insteadOfAck once first when one is scripted. The caller holds mu.
//
// Only the data channel: pacing is a property of the write stream, which
// always goes out there, and a bare call elsewhere waits on its reply, not on
// an acknowledgement. Acking one too would answer a request nobody scripted a
// reply for, and cost it a read that no test expects.
func (d *deviceDouble) autoAck(
	p []byte,
) {
	f, _, err := wire.DecodeFrame(p)
	if err != nil || !f.CarriesData() || len(f.Payload) == 0 {
		return
	}

	name := device.ChannelOf(f)
	if name != device.DataChannel {
		return
	}

	if d.acked == nil {
		d.acked = map[string]int{}
	}

	n := d.acked[name]

	if limit, capped := d.ackLimit[name]; capped && n >= limit {
		if n == limit {
			if frame := d.ackDivert[name]; frame != nil {
				d.autoAcked = append(d.autoAcked, frame)
			}

			d.acked[name] = limit + 1
		}

		return
	}

	d.acked[name] = n + 1

	if d.ackValue == nil {
		d.ackValue = map[string]uint32{}
	}

	value := d.ackValue[name]
	if value == 0 {
		value = ackBase
	}

	value += uint32(len(f.Payload))
	d.ackValue[name] = value

	d.autoAcked = append(d.autoAcked, device.AckFrameFor(name, value))
}

// stopAckingAfter makes a channel go quiet once it has acknowledged n of its
// data frames: a pedal whose receive window has filled.
func (d *deviceDouble) stopAckingAfter(
	channel string,
	n int,
) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.ackLimit == nil {
		d.ackLimit = map[string]int{}
	}

	d.ackLimit[channel] = n
}

// insteadOfAck makes the first data frame past a channel's ack limit answered
// with frame rather than silence: a chunk released by something that was not
// its own acknowledgement, the way a hardware trace showed a control-channel
// frame doing.
func (d *deviceDouble) insteadOfAck(
	channel string,
	frame []byte,
) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.ackDivert == nil {
		d.ackDivert = map[string][]byte{}
	}

	d.ackDivert[channel] = frame
}

// read hands back what is ready, or waits the way a real endpoint does: until
// something is, or until the read's context ends.
func (d *deviceDouble) read(
	ctx context.Context,
	p []byte,
) (int, error) {
	d.mu.Lock()
	d.reads++

	if d.first.IsZero() {
		d.first = time.Now()
	}

	d.mu.Unlock()

	for {
		n, wait, done, err := d.next(p)
		if done {
			return n, err
		}

		if !d.await(ctx, wait) {
			return 0, ctx.Err()
		}
	}
}

// next is what one read gets, if anything, and otherwise how long it may be
// worth waiting before looking again.
func (d *deviceDouble) next(
	p []byte,
) (int, time.Duration, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	held := d.holdFor - time.Since(d.first)

	if held <= 0 {
		if len(d.ready) > 0 {
			frame := d.ready[0]
			d.ready = d.ready[1:]

			return copy(p, frame), 0, true, d.partial
		}

		if len(d.autoAcked) > 0 {
			frame := d.autoAcked[0]
			d.autoAcked = d.autoAcked[1:]

			return copy(p, frame), 0, true, d.partial
		}
	}

	if d.readErr != nil && len(d.sent) >= d.failAfter {
		return 0, 0, true, d.readErr
	}

	if d.noisy != nil {
		return copy(p, d.noisy), 0, true, nil
	}

	return 0, held, false, nil
}

// await waits for something to be ready, for a hold to pass, or for ctx to
// end, and reports whether it is worth looking again.
func (d *deviceDouble) await(
	ctx context.Context,
	held time.Duration,
) bool {
	var hold <-chan time.Time

	if held > 0 {
		timer := time.NewTimer(held)
		defer timer.Stop()

		hold = timer.C
	}

	select {
	case <-ctx.Done():
		return false
	case <-d.wake:
	case <-hold:
	}

	return true
}

// signal wakes a read that is waiting.
func (d *deviceDouble) signal() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// tell makes the device say something unasked, now.
func (d *deviceDouble) tell(
	frames ...[]byte,
) {
	d.mu.Lock()
	d.ready = append(d.ready, frames...)
	d.mu.Unlock()

	d.signal()
}

// frames is everything the session has written so far.
func (d *deviceDouble) frames() [][]byte {
	d.mu.Lock()
	defer d.mu.Unlock()

	return append([][]byte(nil), d.sent...)
}

// pending is how many scripted frames nobody has read yet.
func (d *deviceDouble) pending() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return len(d.replies) + len(d.ready)
}

// readCount is how many times the session read.
func (d *deviceDouble) readCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.reads
}

// ended waits for a session's loop to end on its own, and fails the test
// rather than hanging when it never does.
func ended(
	t *testing.T,
	session *device.Session,
) {
	t.Helper()

	select {
	case <-session.Dead():
	case <-time.After(10 * time.Second):
		t.Fatal("the read loop never ended")
	}
}

// answers scripts a device to reply with each of the given frames in turn,
// one for each frame it is sent.
func answers(
	ctrl *gomock.Controller,
	frames ...[]byte,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.replies = frames

	return d
}

// unasked scripts a device that says each of the given frames straight away,
// before it is asked anything.
func unasked(
	ctrl *gomock.Controller,
	frames ...[]byte,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.ready = frames

	return d
}

// late scripts a device that says nothing until after has passed since it was
// first read, and then replies with each of the given frames in turn.
func late(
	ctrl *gomock.Controller,
	after time.Duration,
	frames ...[]byte,
) *deviceDouble {
	d := answers(ctrl, frames...)
	d.holdFor = after

	return d
}

// readFails scripts a device whose read fails outright.
func readFails(
	ctrl *gomock.Controller,
	err error,
) *deviceDouble {
	return readFailsAfter(ctrl, err, 0)
}

// readFailsAfter scripts a device whose reads fail once it has been sent n
// frames and has nothing left to say.
func readFailsAfter(
	ctrl *gomock.Controller,
	err error,
	n int,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.readErr, d.failAfter = err, n

	return d
}

// writeFails scripts a device whose write fails outright.
func writeFails(
	ctrl *gomock.Controller,
	err error,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.writeErr = err

	return d
}
