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

package wire

import (
	"errors"
	"fmt"
	"slices"
)

// Putting a chain into a preset.
//
// A device lays blocks out on a fixed grid of 20 positions. The first, the
// two in the middle and the last are the input, the split, the join and the
// output, and they are left exactly as the device wrote them. The other 16
// hold blocks or nothing.
//
// The snapshots are written at the same time and cannot be left out. Each of
// the three carries an array of 20 in step with that grid, holding whether
// each position is switched on. A chain written without them recalls the
// wrong blocks the moment anybody presses a snapshot.

// GridSize is how many positions a device lays a chain out on.
const GridSize = 20

// kindEmpty is a grid position holding nothing. The other kinds a position
// declares itself as are named in preset.go.
const kindEmpty = 8

// What a block says it is, in the device's own numbering.
//
// Read off the captures rather than documented: every effect says 1, a
// cabinet on its own says 15, an amp with no cabinet says 17, and an amp
// carrying one says 18.
const (
	ClassEffect = 1
	ClassCab    = 15
	ClassAmp    = 17
	ClassAmpCab = 18
)

// The keys inside a block's body.
const (
	keyClass    = 9
	keyCount    = 2
	keyNamed    = 3
	keyCabHeld  = 23
	keyCabModel = 26
)

// The keys inside a snapshot.
const (
	keySnapBypass = 3
	// snapBypassOn is which of the pair says the position is switched on.
	snapBypassOn = 1
)

// ErrNoRoom is returned when a chain names a position that holds no block.
var ErrNoRoom = errors.New("not a block position")

// NoRoomError names the position and says what is there.
type NoRoomError struct {
	// Position is what was asked for.
	Position int
	// Why says why it cannot hold a block.
	Why string
}

func (e *NoRoomError) Error() string {
	return fmt.Sprintf("position %d cannot hold a block: %s", e.Position, e.Why)
}

func (*NoRoomError) Unwrap() error { return ErrNoRoom }

// Placement is one block to put on the grid.
type Placement struct {
	// Position is where on the grid it goes, counted the way the device
	// lays it out. A footswitch names a block by this number.
	Position int
	// Model is the block's place in the device's model table.
	Model int
	// Values are its parameters, in the order that table names them. Each
	// is a bool, an int64 or a float64.
	Values []any
	// Named is how many of those values the model has names for.
	Named int
	// Enabled is whether the block is switched on.
	Enabled bool
	// Class is what the block says it is: one of the Class constants.
	Class int
	// Cab is the cabinet an amp carries with it, empty when it has none.
	Cab []any
	// CabNamed is how many of those the cabinet names. A paired cabinet
	// sends one more value than that, which is the microphone.
	CabNamed int
	// CabModel is the cabinet's place in the model table, or -1 for none.
	CabModel int
}

// Place writes a chain into a preset, snapshots included.
//
// Every position that can hold a block is written: one the chain names gets
// that block, and one it does not gets emptied. A preset therefore says the
// same thing whatever it held before.
func Place(
	doc *Document,
	blocks []Placement,
) error {
	body, ok := doc.Section(int8(keyTone))
	if !ok {
		return fmt.Errorf("%w: it has no chain", ErrNotADocument)
	}

	at := make(map[int]Placement, len(blocks))

	for _, b := range blocks {
		if err := roomFor(body, b.Position); err != nil {
			return err
		}

		if _, taken := at[b.Position]; taken {
			return &NoRoomError{Position: b.Position, Why: "two blocks want it"}
		}

		at[b.Position] = b
	}

	// Backwards, so a replacement never moves a position not yet written.
	for i := GridSize - 1; i >= 0; i-- {
		start, end, ok := openAt(body, i)
		if !ok {
			continue
		}

		b, named := at[i]

		entry, err := entryFor(b, named)
		if err != nil {
			return err
		}

		body = replaceSpan(body, start, end, entry)
	}

	doc.SetSection(int8(keyTone), body)

	return snapshots(doc, at)
}

// Open lists the grid positions a block may take, in order.
//
// Read off the document rather than assumed. A device decides where it keeps
// the input, the split, the join and the output, and everything left over is
// what a chain can use.
func Open(doc *Document) ([]int, error) {
	body, ok := doc.Section(int8(keyTone))
	if !ok {
		return nil, fmt.Errorf("%w: it has no chain", ErrNotADocument)
	}

	out := []int(nil)

	for i := range GridSize {
		if _, _, ok := openAt(body, i); ok {
			out = append(out, i)
		}
	}

	return out, nil
}

// GridOffset turns a preset's position into a device's.
//
// A preset counts its blocks from zero along a signal path; a device counts
// across a grid that also holds the input, the split, the join and the
// output. The two differ by one, measured against slot 27B of an HX Stomp:
// the preset HX Edit exported puts its six blocks at 1 through 6 and the
// document the device sent puts the same six at 2 through 7.
const GridOffset = 1

// PlaceAsWritten writes a chain where the preset says it goes.
//
// A position past the end of the grid, or one the device keeps its routing
// on, is reported rather than moved: a chain that does not fit is somebody's
// mistake to see, not one to paper over by putting blocks somewhere else.
func PlaceAsWritten(
	doc *Document,
	blocks []Placement,
) error {
	// On a copy: shifting in place would rewrite what the caller handed over,
	// and a second call would shift the same chain twice.
	shifted := slices.Clone(blocks)
	for i := range shifted {
		shifted[i].Position += GridOffset
	}

	return Place(doc, shifted)
}

// roomFor rejects a position that is not the device's to give.
func roomFor(
	body []byte,
	position int,
) error {
	if position < 0 || position >= GridSize {
		return &NoRoomError{
			Position: position,
			Why:      fmt.Sprintf("a device lays out %d", GridSize),
		}
	}

	if _, _, ok := openAt(body, position); !ok {
		return &NoRoomError{
			Position: position,
			Why:      "the device keeps its routing there",
		}
	}

	return nil
}

// openAt returns the bytes a grid position occupies, when a block may take
// it.
//
// The input, the split, the join and the output may not be taken, and where
// they sit is the device's business rather than a fixed layout to assume.
func openAt(
	body []byte,
	position int,
) (int, int, bool) {
	start, end, err := Locate(body, Path{keyBlocks, position})
	if err != nil {
		return 0, 0, false
	}

	entry := body[start:end]

	kindAt, _, err := Locate(entry, Path{keyBlockKind})
	if err != nil {
		return 0, 0, false
	}

	kind, _, err := readKey(entry, kindAt)
	if err != nil || (kind != kindBlock && kind != kindEmpty) {
		return 0, 0, false
	}

	return start, end, true
}

// snapshots writes each snapshot's record of what is switched on.
func snapshots(
	doc *Document,
	at map[int]Placement,
) error {
	body, ok := doc.Section(int8(keySnapshots))
	if !ok {
		return nil
	}

	count, err := countOf(body, Path{keySnapList})
	if err != nil {
		return err
	}

	for snap := range count {
		for i := GridSize - 1; i >= 0; i-- {
			// A snapshot keeping a shorter record than the grid has nothing
			// to say about the rest of it.
			start, end, err := Locate(
				body, Path{keySnapList, snap, keySnapBypass, i, snapBypassOn})
			if err != nil {
				continue
			}

			body = replaceSpan(body, start, end, []byte{boolean(at[i].Enabled)})
		}
	}

	doc.SetSection(int8(keySnapshots), body)

	return nil
}

// entryFor renders one grid position.
//
// A position no chain named is written empty, which is what clears whatever
// the slot held. Asked rather than inferred from the value: model 0 is a
// legitimate index into the device's own table, so a block sitting there with
// nothing set is indistinguishable from a position nobody named.
func entryFor(b Placement, named bool) ([]byte, error) {
	if !named {
		return append(mapHeader(2),
			byte(keyBlockKind), kindEmpty,
			byte(keyBlockBody), codeNil), nil
	}

	params, err := values(b.Values, b.Named)
	if err != nil {
		return nil, err
	}

	cab, err := values(b.Cab, b.CabNamed)
	if err != nil {
		return nil, err
	}

	out := append(mapHeader(2), byte(keyBlockKind), kindBlock)
	out = append(out, byte(keyBlockBody))
	out = append(out, mapHeader(5)...)

	out = append(out, byte(keyClass))
	out = append(out, encodeNumber(b.Class)...)
	out = append(out, byte(keyBypassed))
	out = append(out, boolean(b.Enabled))
	out = append(out, byte(keyParams))
	out = append(out, params...)
	out = append(out, byte(keyPairedCab))
	out = append(out, cab...)

	out = append(out, byte(keyModelRef))
	out = append(out, mapHeader(3)...)
	out = append(out, byte(keyCabHeld), boolean(len(b.Cab) > 0))
	out = append(out, byte(keyModelNum))
	out = append(out, encodeNumber(b.Model)...)
	out = append(out, byte(keyCabModel))

	return append(out, encodeNumber(cabModelOf(b))...), nil
}

// cabModelOf is the cabinet's model number, or the -1 that means none.
func cabModelOf(b Placement) int {
	if len(b.Cab) == 0 {
		return -1
	}

	return b.CabModel
}

// values renders a parameter list the way a block holds one: how many there
// are, how many are named, and the values themselves.
func values(
	list []any,
	named int,
) ([]byte, error) {
	out := append(mapHeader(3), byte(keyCount))
	out = append(out, encodeNumber(len(list))...)
	out = append(out, byte(keyNamed))
	out = append(out, encodeNumber(named)...)
	out = append(out, byte(keyValues))
	out = append(out, arrayHeader(len(list))...)

	for _, v := range list {
		// A value read off a device always encodes: the decoder narrows
		// every one to a bool, an int64 or a float64. Anything else was
		// built here and is a caller's mistake worth reporting rather than
		// writing a hole into somebody's preset.
		raw, err := encodeLike(v, 0)
		if err != nil {
			return nil, err
		}

		out = append(out, raw...)
	}

	return out, nil
}

// countOf reports how many entries an array at path holds.
func countOf(
	body []byte,
	path Path,
) (int, error) {
	start, _, err := Locate(body, path)
	if err != nil {
		return 0, err
	}

	n, _, ok := arrayAt(body, start)
	if !ok {
		return 0, &NoSuchPathError{Path: path, Depth: len(path), Why: "is not an array"}
	}

	return n, nil
}

// encodeNumber renders an integer the way a device writes a small one.
func encodeNumber(v int) []byte {
	return encodeInt(int64(v), 0)
}

// boolean is MessagePack's one-byte true or false.
func boolean(v bool) byte {
	if v {
		return codeTrue
	}

	return codeFalse
}

// arrayHeader counts an array the way the device does.
func arrayHeader(n int) []byte {
	if n < 16 {
		return []byte{byte(0x90 | n)}
	}

	return []byte{codeArray16, byte(n >> 8), byte(n)}
}
