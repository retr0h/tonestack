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

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

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

// formatOf reads the format preset_export was asked for.
//
// Empty is a rig, which is what the tool's schema says leaving it out means.
// Any other name that is not a format is refused, rather than written as a rig
// the agent did not ask for.
func formatOf(
	name string,
) (sdk.Format, error) {
	as := sdk.FormatRig

	if name == "" {
		return as, nil
	}

	if err := as.Set(name); err != nil {
		return "", err
	}

	return as, nil
}

func (h *handlers) devicesList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Attached, error) {
	found, err := locked(ctx, h.pedal, func() (sdk.Attached, error) {
		return h.client.Devices(ctx)
	})
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
	listing, err := onPedal(ctx, h.pedal, func(s Session) (sdk.Listing, error) {
		return s.Presets(ctx, 0)
	})
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

	reading, err := onPedal(ctx, h.pedal, func(s Session) (sdk.Reading, error) {
		return s.Preset(ctx, slot.Address{Slot: n})
	})
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

	// Before the device is claimed: neither a format that does not exist nor
	// a file already at out should cost a USB session. The write refuses a
	// file that appears after this look, too.
	as, err := formatOf(in.As)
	if err != nil {
		return nil, sdk.Written{}, err
	}

	if err := h.mayWrite(in.Out); err != nil {
		return nil, sdk.Written{}, err
	}

	written, err := onPedal(ctx, h.pedal, func(s Session) (sdk.Written, error) {
		return s.Export(ctx, slot.Address{Slot: n}, in.Out, as, h.existing())
	})
	if err != nil {
		return nil, sdk.Written{}, h.refused(in.Out, err)
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

	change, err := onPedal(ctx, h.pedal, func(s Session) (sdk.Change, error) {
		return s.Select(ctx, slot.Address{Slot: n})
	})
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("loaded %s", in.Slot), change, nil
}
