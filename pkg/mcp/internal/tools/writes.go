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
)

// slotsOf reads both ends of a copy or a swap.
func slotsOf(
	in Move,
) (int, int, error) {
	from, err := slotOf(in.From)
	if err != nil {
		return 0, 0, err
	}

	to, err := slotOf(in.To)
	if err != nil {
		return 0, 0, err
	}

	return from, to, nil
}

func (h *handlers) presetImport(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Put,
) (*gomcp.CallToolResult, sdk.Change, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	change, err := onDevice(
		ctx,
		h,
		func() (sdk.Change, error) { return h.client.Import(ctx, sdk.Put{File: in.Preset, Slot: n}) },
	)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("put %s into %s", in.Preset, in.Slot), change, nil
}

func (h *handlers) presetsCopy(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Move,
) (*gomcp.CallToolResult, sdk.Change, error) {
	from, to, err := slotsOf(in)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	change, err := onDevice(
		ctx,
		h,
		func() (sdk.Change, error) { return h.client.Copy(ctx, sdk.Edit{FromSlot: from, ToSlot: to}) },
	)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("copied %s to %s", in.From, in.To), change, nil
}

func (h *handlers) presetsSwap(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Move,
) (*gomcp.CallToolResult, sdk.Change, error) {
	from, to, err := slotsOf(in)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	change, err := onDevice(
		ctx,
		h,
		func() (sdk.Change, error) { return h.client.Swap(ctx, sdk.Edit{FromSlot: from, ToSlot: to}) },
	)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("swapped %s and %s", in.From, in.To), change, nil
}
