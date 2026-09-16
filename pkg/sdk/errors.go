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
	"errors"

	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// EmptySwapError is a swap refused because neither slot holds a preset. It
// matches ErrEmptySlot, and Empty names both slots.
//
// One empty side is a move and is carried out: the preset goes into the empty
// slot and the slot it came from is emptied. Two empty sides is nothing to
// move, so nothing is kept or written.
type EmptySwapError = deviceslots.EmptySwapError

// Errors a caller matches with errors.Is.
//
// Each says what went wrong and nothing about what to do next. A terminal
// runs a command and an agent calls a tool, so the wrapper that knows which
// it is adds the next step.
var (
	// ErrNoSuchBlock reports a model the catalog does not carry.
	ErrNoSuchBlock = catalogview.ErrNotFound
	// ErrNoSuchRecipe reports a rig nobody has written.
	ErrNoSuchRecipe = recipes.ErrNotFound
	// ErrUnknownFormat reports an export asked for a Format that is neither
	// FormatRig nor FormatPreset.
	ErrUnknownFormat = result.ErrUnknownFormat

	// ErrEmptySlot reports a slot on the device holding no preset: one that
	// cannot be the source of a copy, or both sides of a swap. A refused
	// swap carries an EmptySwapError naming the slots.
	ErrEmptySlot = deviceslots.ErrEmptySlot

	// ErrClosed reports a Session method called after Close.
	ErrClosed = errors.New("the session is closed")
	// ErrBus reports a Session the bus ended: a read or a write the bus
	// refused, or the Session's read loop stopping on its own. That Session
	// is finished and nothing reconnects it; a caller who wants the pedal
	// again closes it and opens another.
	ErrBus = device.ErrBus
)
