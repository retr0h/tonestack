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

import "context"

// The scripted device every suite in this package talks to. It answers what
// it was told to answer and records what it was sent, so the protocol can be
// exercised in full with no hardware attached.

// scripted is a scripted answer to whatever is written to it.
type scripted struct {
	// replies are handed back one read at a time.
	replies [][]byte
	// sent is everything the session wrote.
	sent [][]byte
	// writeErr fails every write.
	writeErr error
	// readErr fails every read.
	readErr error
}

func (d *scripted) Write(p []byte) (int, error) {
	if d.writeErr != nil {
		return 0, d.writeErr
	}

	d.sent = append(d.sent, append([]byte(nil), p...))

	return len(p), nil
}

func (d *scripted) ReadContext(_ context.Context, p []byte) (int, error) {
	if d.readErr != nil {
		return 0, d.readErr
	}

	if len(d.replies) == 0 {
		// A device with nothing to say answers with nothing, which is how a
		// drain knows it has finished.
		return 0, nil
	}

	reply := d.replies[0]
	d.replies = d.replies[1:]

	return copy(p, reply), nil
}

// answers scripts a device to reply with each of the given frames in turn.
func answers(frames ...[]byte) *scripted { return &scripted{replies: frames} }
