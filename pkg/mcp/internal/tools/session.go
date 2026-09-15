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

package tools

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// sdkClient is an *sdk.Client whose Open answers the Session the tools hold.
type sdkClient struct {
	*sdk.Client
}

// FromSDK is the Client the tools call, over the SDK's own.
func FromSDK(
	c *sdk.Client,
) Client {
	return sdkClient{Client: c}
}

// Open claims the pedal.
func (c sdkClient) Open(
	ctx context.Context,
) (Session, error) {
	return opened(c.Client.Open(ctx))
}

// opened hands on a Session the SDK opened, or why it could not.
//
// Never a nil *sdk.Session inside a non-nil interface: a holder checking for
// no Session would take that for one, and call it.
func opened(
	s *sdk.Session,
	err error,
) (Session, error) {
	if err != nil {
		return nil, err
	}

	return s, nil
}
