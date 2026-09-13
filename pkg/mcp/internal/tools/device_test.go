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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DeviceTestSuite struct {
	suite.Suite
}

// TestClaim covers waiting for the pedal.
func (s *DeviceTestSuite) TestClaim() {
	tests := []struct {
		name   string
		held   bool
		cancel bool
		err    bool
	}{
		{name: "a free device"},
		{
			// An agent that gives up on a call should not stay queued
			// behind another one that holds the pedal.
			name:   "a held device and a call that gave up",
			held:   true,
			cancel: true,
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			h := &handlers{device: make(chan struct{}, 1)}
			if tt.held {
				h.device <- struct{}{}
			}

			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			release, err := h.claim(ctx)

			if tt.err {
				s.Require().ErrorIs(err, context.Canceled)
				return
			}

			s.Require().NoError(err)
			release()
			s.Empty(h.device)
		})
	}
}

// TestOnDevice covers running a call under the claim, the one place the
// branch each handler used to repeat now lives.
func (s *DeviceTestSuite) TestOnDevice() {
	errCall := errors.New("the call itself failed")

	tests := []struct {
		name     string
		held     bool
		cancel   bool
		callErr  error
		wantErr  error
		wantsRun bool
	}{
		{
			name:     "a free device",
			wantsRun: true,
		},
		{
			// An agent that gives up on a call should not stay queued
			// behind another one that holds the pedal, and the call it
			// gave up on must never run.
			name:    "a held device and a cancelled context",
			held:    true,
			cancel:  true,
			wantErr: context.Canceled,
		},
		{
			name:     "a call that errors",
			callErr:  errCall,
			wantErr:  errCall,
			wantsRun: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			h := &handlers{device: make(chan struct{}, 1)}
			if tt.held {
				h.device <- struct{}{}
			}

			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			ran := false
			got, err := onDevice(ctx, h, func() (int, error) {
				ran = true
				return 7, tt.callErr
			})

			s.Equal(tt.wantsRun, ran)

			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
			} else {
				s.Require().NoError(err)
				s.Equal(7, got)
			}

			if !tt.held {
				// onDevice claimed the device itself, so it released it
				// again whether the call succeeded or failed.
				s.Empty(h.device)
			}
		})
	}
}

func TestDeviceTestSuite(t *testing.T) {
	suite.Run(t, new(DeviceTestSuite))
}
