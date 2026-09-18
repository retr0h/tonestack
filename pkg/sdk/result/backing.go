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
package result

// Backing is which records back a rig, and whether they were made when its
// gear was.
//
// A rig's audio evidence is measured from records. The rig claims gear for a
// period, and a record made outside that period measures other gear: Paul
// McCartney's rig describes an Acoustic 360, and one of his measured records
// was cut two years before Acoustic built one. The figures are honest, the
// rig is honest, and the join between them is wrong.
type Backing struct {
	// ID is the rig.
	ID string
	// Era is the years the rig claims, as the rig states them.
	Era string
	// From and To are those years as numbers. Zero when the rig states none,
	// which is itself worth reporting: nothing can be held to an era nobody
	// wrote down.
	From int
	To   int
	// Records are what was measured for it, in the order the manifest names
	// them.
	Records []Record
	// Misnamed are track names a `played.records` entry gives that no
	// manifest has.
	//
	// The list exists so a figure can be read against the instrument that
	// made it, and a name matching nothing joins to nothing. It fails exactly
	// like the directory-name join it borrows: silently, and looking like an
	// instrument nobody has attributed yet.
	Misnamed []string
	// NoRig says these records sit in a directory no rig answers to.
	//
	// A rig is joined to its records by the directory being named for it, and
	// a directory named anything else is silently measured by nobody. Records
	// arriving before the rig that will use them is the ordinary reason; a
	// typo is the other one, and it looks identical until this says so.
	NoRig bool
	// Direct counts the chain entries whose signal never met a microphone.
	//
	// The second kind of wrong-era mistake. A record made in the right years
	// can still have been made in another room: five of the nine rigs here
	// measure a signal that went to the desk, and every one of them ends in a
	// cabinet. The cabinet is not wrong, since a preset with none into a PA
	// is not the sound either, but a figure measured off that record was not
	// shaped by it.
	Direct int
	// Both counts the entries that went to the desk and through a microphone
	// at once, which is a third answer rather than a hedge. Jaco Pastorius
	// took "a little bit of both, the highs and lows".
	Both int
	// Captured counts the entries that say anything at all about how they
	// reached the tape. Zero means nobody has established it, which is not
	// the same as miked and must not read as it.
	Captured int
	// Stage counts the chain entries whose only evidence is a tour.
	//
	// A rig rundown photographs a backline and the corpus measures records.
	// Both are honest and they are not the same rig.
	Stage int
}

// Stated says whether the rig gives years its records can be held to.
func (b Backing) Stated() bool { return b.From != 0 && b.To != 0 }

// Outside counts the records made outside the era the rig claims.
func (b Backing) Outside() int {
	out := 0

	for _, r := range b.Records {
		if r.Outside {
			out++
		}
	}

	return out
}

// Record is one record measured for a rig.
type Record struct {
	// Track is what the manifest calls it.
	Track string
	// Year is when it was made.
	Year int
	// Outside says the year falls outside the rig's era.
	Outside bool
}
