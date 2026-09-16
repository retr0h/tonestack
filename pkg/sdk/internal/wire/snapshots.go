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

// Putting somebody else's snapshots into a preset.
//
// place.go writes every snapshot's record of what is switched on from the
// chain itself, which is right for a chain nobody has snapshots for: all three
// then recall the same sound. A preset that came from somewhere with snapshots
// of its own has three different sounds to say, and this is what says them.

// Snapshot is what one snapshot recalls, as a caller describes it.
//
// Every field is optional, and what a caller leaves out keeps whatever the
// preset being written into already holds. A preset is written into one the
// device wrote, which ships three snapshots of its own, so a source saying
// nothing about a tempo is not asking for a tempo of zero.
type Snapshot struct {
	// Name is what the device shows for it.
	Name *string
	// Tempo is what it recalls, in beats per minute.
	Tempo *float64
	// LED is the colour the footswitch lights, as the device numbers them.
	LED *int
	// Valid is whether the device considers the snapshot set up.
	Valid *bool
	// On is which grid positions it switches on, counted the way the device
	// lays a chain out. A position it does not name is left as it is.
	On map[int]bool
}

// PlaceSnapshots writes what each snapshot recalls.
//
// Over the snapshots the preset already has and no further. A device ships
// three, and a source carrying more is saying something this preset has
// nowhere to put: those go where nothing is, which writes nothing.
//
// The four positions a chain may not take are left alone, for the reason
// place.go gives — a device keeps its input, its split, its join and its
// output on the same grid a snapshot records, and switching those off recalls
// a chain with its routing bypassed.
//
// Nothing here fails. A preset keeping no snapshots has none to overwrite, and
// one with no chain cannot say which positions are the routing's, so both are
// left as they are: PlaceAsWritten refuses the second before this ever sees
// it.
func PlaceSnapshots(
	doc *Document,
	snaps []Snapshot,
) {
	if len(snaps) == 0 {
		return
	}

	body, ok := doc.Section(int8(keySnapshots))
	if !ok {
		return
	}

	tone, ok := doc.Section(int8(keyTone))
	if !ok {
		return
	}

	for at, snap := range snaps {
		body = placeSnapshot(body, tone, at, snap)
	}

	doc.SetSection(int8(keySnapshots), body)
}

// placeSnapshot writes one snapshot's own fields and its record of the grid.
func placeSnapshot(
	body []byte,
	tone []byte,
	at int,
	snap Snapshot,
) []byte {
	if snap.Name != nil {
		body = setAt(body, path{keySnapList, at, keySnapName}, *snap.Name)
	}

	if snap.Tempo != nil {
		body = setAt(body, path{keySnapList, at, keySnapTempo}, *snap.Tempo)
	}

	if snap.LED != nil {
		body = setAt(body, path{keySnapList, at, keySnapLED}, *snap.LED)
	}

	if snap.Valid != nil {
		body = setAt(body, path{keySnapList, at, keySnapValid}, *snap.Valid)
	}

	// Each write is located afresh against the bytes the one before it left,
	// so the order these go in does not matter. A position the chain may not
	// take belongs to the routing and keeps what it holds.
	for i := range gridSize {
		on, named := snap.On[i]
		if !named {
			continue
		}

		if _, _, ok := openAt(tone, i); !ok {
			continue
		}

		body = setAt(body, path{keySnapList, at, keySnapBypass, i, snapBypassOn}, on)
	}

	return body
}

// setAt writes one value where a path says, in the encoding the device gave
// whatever was there.
//
// A path this preset does not carry is left alone. A snapshot that keeps no
// tempo, or a fourth snapshot on a device that holds three, is nothing to
// overwrite rather than a caller's mistake.
func setAt(
	body []byte,
	at path,
	v any,
) []byte {
	start, end, err := locate(body, at)
	if err != nil {
		return body
	}

	// A Snapshot carries a string, a float, an integer and a boolean, and
	// encodeLike takes all four, so this cannot fail.
	raw, _ := encodeLike(v, body[start])

	return replaceSpan(body, start, end, raw)
}
