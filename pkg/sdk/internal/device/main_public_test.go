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
	"testing"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// TestMain shortens every wait this package spends on hardware, since nothing
// here is talking to any.
//
// The ordering is the one a device has: a commit outlasts a reply, and a
// write is waited on for the commit budget. A test that needs a different
// figure sets its own and puts this one back.
func TestMain(m *testing.M) {
	*device.ReplyBudget = 50 * time.Millisecond
	*device.DrainBudget = 50 * time.Millisecond
	*device.CommitBudget = 500 * time.Millisecond
	*device.FlashBudget = 0
	*device.CloseBudget = 500 * time.Millisecond

	m.Run()
}
