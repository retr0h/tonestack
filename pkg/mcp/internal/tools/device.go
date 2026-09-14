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
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// claim takes the pedal for one call, or gives up when the call does.
//
// A USB editor interface serves one session. Two calls claiming it at once
// is the failure docs/protocol.md warns leaves a pedal needing a power cycle.
func (h *handlers) claim(
	ctx context.Context,
) (func(), error) {
	select {
	case h.device <- struct{}{}:
		return func() { <-h.device }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for the device: %w", ctx.Err())
	}
}

// onDevice runs call while holding the pedal, or gives up when ctx does.
func onDevice[T any](
	ctx context.Context,
	h *handlers,
	call func() (T, error),
) (T, error) {
	release, err := h.claim(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	defer release()

	return call()
}

// slotOf reads a slot the way the pedal labels it.
func slotOf(
	label string,
) (int, error) {
	var n int
	if err := slot.NewValue(&n).Set(label); err != nil {
		return 0, err
	}

	return n, nil
}

func (h *handlers) devicesList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Attached, error) {
	found, err := onDevice(ctx, h, func() (sdk.Attached, error) { return h.client.Devices(ctx) })
	if err != nil {
		return nil, sdk.Attached{}, err
	}

	return said("%d attached", len(found.Devices)), found, nil
}

func (h *handlers) presetsList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Listing, error) {
	listing, err := onDevice(
		ctx, h, func() (sdk.Listing, error) { return h.client.Presets(ctx, sdk.Where{}) },
	)
	if err != nil {
		return nil, sdk.Listing{}, err
	}

	return said("%d of %d slots hold a preset", listing.Used(), len(listing.Slots)), listing, nil
}

func (h *handlers) presetShow(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Slot,
) (*gomcp.CallToolResult, Shown, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, Shown{}, err
	}

	reading, err := onDevice(
		ctx, h, func() (sdk.Reading, error) { return h.client.Preset(ctx, sdk.Read{Slot: n}) },
	)
	if err != nil {
		return nil, Shown{}, err
	}

	out := Shown{Name: reading.Name, Rig: reading.Rig, Answer: reading.Answer}

	return said("%s holds %s", in.Slot, reading.Name), out, nil
}

func (h *handlers) presetExport(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Export,
) (*gomcp.CallToolResult, sdk.Written, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, sdk.Written{}, err
	}

	written, err := onDevice(ctx, h, func() (sdk.Written, error) {
		return h.client.Export(ctx, sdk.Export{Slot: n, OutputPath: in.Out, As: in.As})
	})
	if err != nil {
		return nil, sdk.Written{}, err
	}

	return said("wrote %s from %s", written.Path, in.Slot), written, nil
}

func (h *handlers) presetSelect(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Slot,
) (*gomcp.CallToolResult, sdk.Change, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	change, err := onDevice(
		ctx, h, func() (sdk.Change, error) { return h.client.Select(ctx, sdk.Read{Slot: n}) },
	)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("loaded %s", in.Slot), change, nil
}
