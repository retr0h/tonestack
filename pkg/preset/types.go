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
// Package preset reads and writes Line 6 .hlx preset files.
//
// The format has no published schema; everything here was established by
// reading real presets. See docs/preset-format.md.
package preset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// Schema is the value every preset carries in its schema field.
const Schema = "L6Preset"

// Version is the preset schema version this package writes.
const Version = 6

// Attribute keys. Everything else in a block object is a parameter.
const (
	attrModel    = "@model"
	attrPosition = "@position"
	attrEnabled  = "@enabled"
	attrPath     = "@path"
	attrStereo   = "@stereo"
	attrType     = "@type"
)

// Document is a preset file as it appears on disk.
//
// Fields this package does not model are preserved verbatim in Rest, so a
// preset read and written again keeps whatever the device put there.
type Document struct {
	Schema  string          `json:"schema"`
	Version int             `json:"version"`
	Data    Data            `json:"data"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}

// Data is the preset's payload.
type Data struct {
	Device        int             `json:"device"`
	DeviceVersion FlexInt         `json:"device_version"`
	Meta          DataMeta        `json:"meta"`
	Tone          map[string]Tone `json:"tone"`
}

// DataMeta names the preset. The name a person sees lives here, not in the
// document's top-level meta.
type DataMeta struct {
	// Name is what a player sees, and the only field this package has an
	// opinion about.
	Name string
	// Rest is everything else, kept exactly as it arrived.
	//
	// Presets in the wild carry fields nobody documented — song, band,
	// author, tnid, an appVersion spelled two different ways. Modelling a
	// fixed set drops the rest, which silently rewrites somebody's preset.
	// A file this tool wrote should differ from the original only where
	// somebody asked it to.
	Rest map[string]json.RawMessage
}

// nameKey is the one metadata field this package reads.
const nameKey = "name"

// UnmarshalJSON keeps every field, modelled or not.
func (m *DataMeta) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &m.Rest); err != nil {
		return fmt.Errorf("decoding preset metadata: %w", err)
	}

	if raw, ok := m.Rest[nameKey]; ok {
		if err := json.Unmarshal(raw, &m.Name); err != nil {
			return fmt.Errorf("decoding preset name: %w", err)
		}
	}

	return nil
}

// MarshalJSON writes back what arrived, with the name as it now stands.
func (m DataMeta) MarshalJSON() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(m.Rest)+1)
	for k, v := range m.Rest {
		out[k] = v
	}

	// A string always marshals, so there is no failure to report.
	name, _ := json.Marshal(m.Name)
	out[nameKey] = name

	return json.Marshal(out)
}

// Tone is one entry under the tone object: a processor's blocks, a snapshot,
// or a controller assignment. Only processors are modelled; the rest is held
// as raw JSON and written back unchanged.
type Tone map[string]json.RawMessage

// Block is one entry in a processor.
//
// Attributes are @-prefixed; everything else is a parameter. Params holds
// them in the union that keeps a float from being written where the device
// expects an enum.
type Block struct {
	Model catalog.ModelID
	// Slot is the number in the block's own key — block5 is slot 5.
	//
	// Independent of Position: a preset can hold block5 whose @position is 6.
	// Deriving one from the other moves blocks around a preset that nobody
	// asked to change.
	Slot     int
	Position int
	Enabled  bool
	Path     int
	Stereo   bool
	Type     int
	Params   map[string]catalog.ParamValue
	// Attrs holds every @-prefixed attribute except the three modelled
	// above, exactly as it arrived.
	//
	// A device owns these — @path, @stereo, @type, @trails,
	// @no_snapshot_bypass — and a preset this tool rewrote should differ from
	// the original only where somebody asked it to.
	Attrs map[string]json.RawMessage
}

// FlexInt is an integer that tolerates being written as a string.
//
// Devices always write device_version as a number. Hand-made templates in the
// wild do not, and refusing to read one because of a field nothing depends on
// would be pedantry rather than correctness.
type FlexInt struct {
	Value int
	// Raw is the literal exactly as it arrived, set whenever the value came
	// in as a string.
	//
	// Kept rather than reconstructed because the string forms do not survive
	// being parsed and reprinted: "0.00" comes back as "0", and rewriting a
	// preset that way changes a file nobody asked to change.
	Raw json.RawMessage
}

// MarshalJSON writes the value back in the form it arrived in.
func (f FlexInt) MarshalJSON() ([]byte, error) {
	if len(f.Raw) > 0 {
		return f.Raw, nil
	}

	return json.Marshal(f.Value)
}

// UnmarshalJSON accepts a number or a string holding one.
func (f *FlexInt) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		f.Value = n

		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("device_version is neither a number nor a string: %w", err)
	}

	if s == "" {
		// Cloned: encoding/json lends the buffer and reuses it, and what is
		// kept here is written back out verbatim.
		f.Raw = bytes.Clone(b)

		return nil
	}

	// "0.00" appears in hand-made templates, so parse as a float and take
	// the integer part rather than rejecting the file.
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("device_version %q is not a number: %w", s, err)
	}

	f.Value = int(v)
	f.Raw = bytes.Clone(b)

	return nil
}
