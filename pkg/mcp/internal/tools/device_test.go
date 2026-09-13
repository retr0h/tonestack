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
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
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

// TestHandlersGiveUp covers the branch inside each handler that a cancelled
// context takes when another call already holds the pedal: reachable only
// from inside the package, since a real call through the server has no way to
// hold the device and cancel the same context at once.
func (s *DeviceTestSuite) TestHandlersGiveUp() {
	tests := []struct {
		name string
		call func(h *handlers, ctx context.Context) error
	}{
		{
			name: "devices_list",
			call: func(h *handlers, ctx context.Context) error {
				_, _, err := h.devicesList(ctx, &gomcp.CallToolRequest{}, None{})
				return err
			},
		},
		{
			name: "presets_list",
			call: func(h *handlers, ctx context.Context) error {
				_, _, err := h.presetsList(ctx, &gomcp.CallToolRequest{}, None{})
				return err
			},
		},
		{
			name: "preset_show",
			call: func(h *handlers, ctx context.Context) error {
				_, _, err := h.presetShow(ctx, &gomcp.CallToolRequest{}, Slot{Slot: "01A"})
				return err
			},
		},
		{
			name: "preset_export",
			call: func(h *handlers, ctx context.Context) error {
				_, _, err := h.presetExport(ctx, &gomcp.CallToolRequest{}, Export{Slot: "01A", Out: "a.yaml"})
				return err
			},
		},
		{
			name: "preset_select",
			call: func(h *handlers, ctx context.Context) error {
				_, _, err := h.presetSelect(ctx, &gomcp.CallToolRequest{}, Slot{Slot: "01A"})
				return err
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			h := &handlers{device: make(chan struct{}, 1)}
			h.device <- struct{}{}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := tt.call(h, ctx)

			s.Require().ErrorIs(err, context.Canceled)
		})
	}
}

func TestDeviceTestSuite(t *testing.T) {
	suite.Run(t, new(DeviceTestSuite))
}
