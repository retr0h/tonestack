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
	"time"

	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// The scripted device every suite in this package talks to. It answers what
// it was told to answer and records what it was sent, so the protocol can be
// exercised in full with no hardware attached.
//
// The sender and receiver it presents are generated mocks (device.Mocksender
// and device.Mockreceiver); deviceDouble only groups them with the closures
// that give them their behaviour, the way CONTRIBUTING's "Test doubles"
// section asks of a hand-rolled type standing in for a generated one.

// deviceDouble is a sender and receiver mock scripted to answer whatever is
// written to it.
type deviceDouble struct {
	// out and in are what a session talks to. NewTestSession takes them
	// directly.
	out *device.Mocksender
	in  *device.Mockreceiver

	// replies are handed back one read at a time.
	replies [][]byte
	// sent is everything the session wrote.
	sent [][]byte
	// writeErr fails every write.
	writeErr error
	// readErr fails every read after the first readsOK of them.
	readErr error
	readsOK int
	// reads is how many times the session read.
	reads int
	// onWrite runs after every write the device took, so a test can stop
	// waiting partway through a message.
	onWrite func()
	// noisy is handed back on every read once replies run out: a device that
	// never goes quiet.
	noisy []byte
	// pause is how long a quiet read takes, cut short when its context ends:
	// a device with nothing to say, read the way a real endpoint is read.
	pause time.Duration
	// holdFor keeps the replies back until that long after the first read: a
	// device that answers, but late.
	holdFor time.Duration
	first   time.Time
}

// newDeviceDouble wires a sender and a receiver mock to a device's
// behaviour. Mutating a field afterwards changes what the mocks do on the
// next call, the same as the hand-rolled scripted type's fields did.
func newDeviceDouble(
	ctrl *gomock.Controller,
) *deviceDouble {
	d := &deviceDouble{}

	d.out = device.NewMocksender(ctrl)
	d.out.EXPECT().Write(gomock.Any()).DoAndReturn(d.write).AnyTimes()

	d.in = device.NewMockreceiver(ctrl)
	d.in.EXPECT().ReadContext(gomock.Any(), gomock.Any()).DoAndReturn(d.read).AnyTimes()

	return d
}

// write records what the session sent, and fails it when writeErr is set.
func (d *deviceDouble) write(
	p []byte,
) (int, error) {
	if d.writeErr != nil {
		return 0, d.writeErr
	}

	d.sent = append(d.sent, append([]byte(nil), p...))

	if d.onWrite != nil {
		d.onWrite()
	}

	return len(p), nil
}

// read hands back the next scripted reply, and fails it once readsOK of them
// have gone out.
func (d *deviceDouble) read(
	ctx context.Context,
	p []byte,
) (int, error) {
	d.reads++

	if d.first.IsZero() {
		d.first = time.Now()
	}

	if d.readErr != nil && d.reads > d.readsOK {
		return 0, d.readErr
	}

	if len(d.replies) == 0 && d.noisy != nil {
		return copy(p, d.noisy), nil
	}

	held := d.holdFor > 0 && time.Since(d.first) < d.holdFor

	if len(d.replies) == 0 || held {
		// A device with nothing to say answers with nothing inside its read
		// window: context.DeadlineExceeded, what IOKit's readUntil returns
		// once its own timeout elapses.
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(d.pause):
			return 0, context.DeadlineExceeded
		}
	}

	reply := d.replies[0]
	d.replies = d.replies[1:]

	return copy(p, reply), nil
}

// answers scripts a device to reply with each of the given frames in turn.
func answers(
	ctrl *gomock.Controller,
	frames ...[]byte,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.replies = frames

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
	// A quiet read that returned at once would spin the wait.
	d.pause = time.Millisecond

	return d
}

// readFails scripts a device whose read fails outright.
func readFails(
	ctrl *gomock.Controller,
	err error,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.readErr = err

	return d
}

// readFailsAfter scripts a device whose reads fail once n of them have
// succeeded.
func readFailsAfter(
	ctrl *gomock.Controller,
	err error,
	n int,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.readErr, d.readsOK = err, n

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

// paused scripts a device that takes wait before answering a quiet read.
func paused(
	ctrl *gomock.Controller,
	wait time.Duration,
) *deviceDouble {
	d := newDeviceDouble(ctrl)
	d.pause = wait

	return d
}
