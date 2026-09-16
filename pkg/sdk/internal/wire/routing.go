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

// Putting somebody else's routing into a preset.
//
// place.go writes the sixteen positions a chain may take and refuses the other
// four, which is what keeps a chain from switching off the device's own input,
// split, join and output. That leaves those four holding whatever the preset
// being written into came with, so a preset built from a file routes the way
// the blank routes rather than the way the file says.
//
// A device wraps a chain in six entries and lays them out in the same array as
// the blocks: the first holds the input, the last the output, and the two in
// the middle each carry a pair, the second input beside the split and the
// second output beside the join.

// Routing is one entry either side of a chain, as a caller describes it.
//
// Every field is optional and what a caller leaves out keeps whatever the
// preset already holds, for the reason Snapshot gives: a preset is written
// into one a device wrote, and saying nothing about a field is not asking for
// a zero.
type Routing struct {
	// Slot names which entry this is, the way a preset names it: inputA,
	// inputB, outputA, outputB, split, join. Anything else is ignored.
	Slot string
	// Select is which input or output this is. Only inputs and outputs carry
	// one; a device knows which are its own.
	Select *int
	// Model is where a split or a join sits in the device's model table. An
	// input and an output carry none.
	Model *int
	// Position is where a split or a join sits in the layout.
	Position *int
	// Enabled is whether a split or a join is switched on.
	Enabled *bool
	// Values are the parameters, in the order the model table names them.
	// Nothing keeps the ones already there, which is not the same as an empty
	// list: the blank's second input really does carry none.
	Values *[]any
	// Named is how many of those the model has names for.
	Named int
}

// slotAt says where in a chain entry one routing slot is written.
//
// The two middle entries each hold a pair, so four of the six sit one level
// further in than the other two.
var slotAt = map[string]struct {
	// kind is the chain entry that carries this slot.
	kind int
	// inner is the key the slot sits under inside that entry, when the entry
	// carries a pair rather than being the slot itself.
	inner int
	// nested says whether inner means anything.
	nested bool
	// selects is the key holding which input or output this is, or zero when
	// the slot is a split or a join and carries none.
	selects int
}{
	"inputA":  {kind: kindInput, selects: keyInputSelect},
	"outputA": {kind: kindOutput, selects: keyOutputSelect},
	"inputB":  {kind: kindSplit, inner: keySplitInput, nested: true, selects: keyInputSelect},
	"split":   {kind: kindSplit, inner: keySplitBlock, nested: true},
	"outputB": {kind: kindJoin, inner: keyJoinOutput, nested: true, selects: keyOutputSelect},
	"join":    {kind: kindJoin, inner: keyJoinBlock, nested: true},
}

// BlankRouting is what an unused slot wraps a chain in.
//
// Which entries a device lays out, and how many values each one carries. A
// device sends fewer than a model names, so this is the only honest source of
// those lengths for anything building a preset into Blank.
//
// The errors are discarded for the reason Blank gives: it is embedded in this
// binary and read by a test, so it cannot fail to decode.
func BlankRouting() []DeviceRouting {
	doc, _ := Blank()
	got, _ := DecodePreset(doc.Encode())

	return got.Routing
}

// PlaceRouting writes what a device wraps a chain in.
//
// Over the entries the preset already has. Each one is found by the kind it
// declares itself to be rather than by where this blank happens to keep it,
// because which position holds the split is the device's business and not a
// layout to assume.
//
// A preset with no chain has nowhere to put any of this and is left alone.
func PlaceRouting(
	doc *Document,
	routes []Routing,
) error {
	if len(routes) == 0 {
		return nil
	}

	body, ok := doc.Section(int8(keyTone))
	if !ok {
		return nil
	}

	var err error

	for _, r := range routes {
		if body, err = placeRoute(body, r); err != nil {
			return err
		}
	}

	doc.SetSection(int8(keyTone), body)

	return nil
}

// placeRoute writes one routing entry's fields.
func placeRoute(
	body []byte,
	r Routing,
) ([]byte, error) {
	where, ok := slotAt[r.Slot]
	if !ok {
		return body, nil
	}

	found, ok := kindAt(body, where.kind)
	if !ok {
		return body, nil
	}

	base := path{keyBlocks, found, keyBlockBody}
	if where.nested {
		base = under(base, where.inner)
	}

	if r.Select != nil && where.selects != 0 {
		body = setAt(body, under(base, where.selects), *r.Select)
	}

	if r.Model != nil {
		body = setAt(body, under(base, keyFlowModel), *r.Model)
	}

	if r.Position != nil {
		body = setAt(body, under(base, keyFlowPosition), *r.Position)
	}

	if r.Enabled != nil {
		body = setAt(body, under(base, keyFlowEnabled), *r.Enabled)
	}

	if r.Values == nil {
		return body, nil
	}

	// The whole parameter map, not the values inside it: a split the file
	// names is not always the one the blank carries, and a different model
	// has a different number of parameters.
	raw, err := values(*r.Values, r.Named)
	if err != nil {
		return nil, err
	}

	at := under(base, keyFlowParams)

	start, end, err := locate(body, at)
	if err != nil {
		// A preset keeping fewer entries than a caller describes is the
		// device's business rather than a caller's mistake, the same as setAt.
		return body, nil
	}

	return replaceSpan(body, start, end, raw), nil
}

// kindAt finds the grid position holding one kind of chain entry.
//
// By what the entry says it is. A device is free to lay its routing out where
// it likes, and this blank's layout is one device's rather than a rule.
func kindAt(
	body []byte,
	want int,
) (int, bool) {
	found, ok := 0, false

	for i := range gridSize {
		// An entry whose kind will not read says nothing about itself, and no
		// device has sent one. Asking rather than assuming costs nothing.
		if kind, read := kindOf(body, i); read && kind == want {
			found, ok = i, true

			break
		}
	}

	return found, ok
}

// kindOf reads what one grid position declares itself to be.
func kindOf(
	body []byte,
	at int,
) (int, bool) {
	start, end, err := locate(body, path{keyBlocks, at})
	if err != nil {
		return 0, false
	}

	entry := body[start:end]

	keyAt, _, err := locate(entry, path{keyBlockKind})
	if err != nil {
		return 0, false
	}

	kind, _, err := readKey(entry, keyAt)
	if err != nil {
		return 0, false
	}

	return kind, true
}

// under returns a fresh path one step further in.
//
// Fresh because appending to a shared path hands every write the same backing
// array, and each would overwrite the step the one before it took.
func under(
	base path,
	keys ...int,
) path {
	out := make(path, 0, len(base)+len(keys))
	out = append(out, base...)

	return append(out, keys...)
}
