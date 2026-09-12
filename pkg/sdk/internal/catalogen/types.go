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
// Package catalogen builds a device catalog from Line 6's own model
// definitions.
//
// HX Edit ships Line 6's complete model data as JSON inside its app bundle:
// every model's identifier, name, DSP cost, and every parameter's real range
// and default. Joined with resources/schemas/gear-map.json — which says what each model
// emulates — that is everything needed to place a block in a chain.
//
// The data belongs to Line 6 and reaches us only through a licensed HX Edit
// installation, so the catalog is generated locally and never redistributed. A
// machine without HX Edit cannot build one, and Build says so plainly rather
// than falling back to guesswork.
package catalogen

import "encoding/json"

// Options controls what is built and from where.
type Options struct {
	// ResourcesDir is HX Edit's Contents/Resources directory.
	ResourcesDir string
	// SourceName names the application the models came from. The version is
	// read from the bundle and appended to it.
	SourceName string
	// GearMapPath is resources/schemas/gear-map.json.
	GearMapPath string
	// DeviceID is the integer a preset for the target device carries in
	// data.device. Only models the device supports are included.
	DeviceID int
	// DeviceName is how Line 6 markets the device, e.g. "HX Stomp".
	DeviceName string
	// SchemaVersion is the preset schema version to record.
	SchemaVersion int
	// OutputPath is where the catalog is written.
	OutputPath string
}

// wireModel is one entry in a Line 6 .models file.
//
// Numeric fields are held as raw JSON because their type depends on
// valueType: a bool parameter reports false and true as its bounds, which
// carries no information and does not fit a float.
type wireModel struct {
	SymbolicID string       `json:"symbolicID"`
	Name       string       `json:"name"`
	CabLink    string       `json:"cablink"`
	Load       *float64     `json:"load"`
	LoadStereo *float64     `json:"load_stereo"`
	Stereo     *bool        `json:"stereo"`
	Devices    []wireDevice `json:"devices"`
	Params     []wireParam  `json:"params"`
}

// wireDevice names a device that supports a model, with the firmware it
// arrived in.
type wireDevice struct {
	ID int `json:"id"`
}

// wireParam is one parameter of a wireModel.
type wireParam struct {
	SymbolicID  string          `json:"symbolicID"`
	Name        string          `json:"name"`
	ValueType   int             `json:"valueType"`
	DisplayType string          `json:"displayType"`
	Min         json.RawMessage `json:"min"`
	Max         json.RawMessage `json:"max"`
	Default     json.RawMessage `json:"default"`
}

// gearMap is resources/schemas/gear-map.json.
type gearMap struct {
	Models map[string]gearEntry `json:"models"`
}

// gearEntry says what one model emulates.
type gearEntry struct {
	Name        string `json:"name"`
	Family      string `json:"family"`
	Subcategory string `json:"subcategory"`
	BasedOn     string `json:"based_on"`
}
