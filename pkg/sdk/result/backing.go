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
	// NoRig says these records sit in a directory no rig answers to.
	//
	// A rig is joined to its records by the directory being named for it, and
	// a directory named anything else is silently measured by nobody. Records
	// arriving before the rig that will use them is the ordinary reason; a
	// typo is the other one, and it looks identical until this says so.
	NoRig bool
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
