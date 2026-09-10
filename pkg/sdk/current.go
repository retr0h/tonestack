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
)

// opReadCurrent reads the preset a device is playing.
const opReadCurrent = 22

// ReadCurrent fetches the preset the device has loaded.
//
// The edit buffer rather than a slot: what somebody is hearing, including
// whatever they have changed since it was loaded. A stored slot is what
// ReadPreset answers with, and the two differ — a loaded document carries the
// firmware build string a stored one does not.
func (s *session) ReadCurrent(ctx context.Context) ([]byte, error) {
	resp, err := s.Call(ctx, channelData, opReadCurrent, nil)
	if err != nil {
		return nil, err
	}

	return document(resp.Result)
}
