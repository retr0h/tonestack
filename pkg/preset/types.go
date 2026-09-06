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
	Name         string `json:"name"`
	Application  string `json:"application,omitempty"`
	AppVersion   int    `json:"appversion,omitempty"`
	BuildSHA     string `json:"build_sha,omitempty"`
	ModifiedDate int64  `json:"modifieddate,omitempty"`
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
	Model    catalog.ModelID
	Position int
	Enabled  bool
	Path     int
	Stereo   bool
	Type     int
	Params   map[string]catalog.ParamValue
}

// FlexInt is an integer that tolerates being written as a string.
//
// Devices always write device_version as a number. Hand-made templates in the
// wild do not, and refusing to read one because of a field nothing depends on
// would be pedantry rather than correctness.
type FlexInt int

// UnmarshalJSON accepts a number or a string holding one.
func (f *FlexInt) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*f = FlexInt(n)

		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("device_version is neither a number nor a string: %w", err)
	}

	if s == "" {
		return nil
	}

	// "0.00" appears in hand-made templates, so parse as a float and take
	// the integer part rather than rejecting the file.
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("device_version %q is not a number: %w", s, err)
	}

	*f = FlexInt(int(v))

	return nil
}
