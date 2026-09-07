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

// Package catalog describes what a Helix device can do: which blocks exist,
// what parameters each accepts, and how much DSP each costs. It is generated
// from Line 6's own model definitions plus a corpus of real presets, because
// the preset format has no published schema.
package catalog

// ModelID is a Line 6 internal model identifier, such as "HD2_AmpAmpegSVT".
type ModelID string

// Category groups blocks by what they do in a signal chain.
type Category string

// The block categories this catalog distinguishes.
const (
	CategoryAmp    Category = "amp"
	CategoryCab    Category = "cab"
	CategoryDrive  Category = "drive"
	CategoryComp   Category = "comp"
	CategoryDelay  Category = "delay"
	CategoryReverb Category = "reverb"
	CategoryEQ     Category = "eq"
	CategoryMod    Category = "mod"
	// CategoryGate is a noise gate. Dynamics, but not a compressor: counting
	// the two together overstates how often players compress.
	CategoryGate Category = "gate"
	// CategoryWah is a wah or auto-wah.
	CategoryWah Category = "wah"
	// CategoryPitch is pitch shifting, harmony and synthesis.
	CategoryPitch Category = "pitch"
	// CategoryFilter is a filter or envelope follower.
	CategoryFilter Category = "filter"
	// CategoryUtility is plumbing rather than tone — volume, gain, sends,
	// loopers, the input and output blocks. Nobody chooses one for how it
	// sounds, so they are excluded from anything measuring what a chain is
	// made of.
	CategoryUtility Category = "utility"
	CategoryOther   Category = "other"
)

// Provenance records how a catalog entry came to be known, and how far it can
// be trusted. It gates delivery: nothing whose DSP cost is ProvAssumed may
// reach a user.
type Provenance string

// How a catalog entry was learned, most trustworthy first.
const (
	// ProvOfficial came from Line 6's own model definitions, which ship as
	// JSON inside HX Edit. Ranges, defaults and DSP costs are stated by the
	// vendor rather than inferred, and are authoritative.
	ProvOfficial Provenance = "official"
	// ProvMeasured was swept on real hardware. Ranges are trustworthy.
	ProvMeasured Provenance = "measured"
	// ProvObserved was seen in exported presets. Identifiers and parameter
	// keys are trustworthy; ranges are only what the corpus happened to
	// contain, so they are lower bounds rather than true bounds.
	ProvObserved Provenance = "observed"
	// ProvInherited came from a third-party catalog and is unverified.
	ProvInherited Provenance = "inherited"
	// ProvAssumed is a guess. It never ships.
	ProvAssumed Provenance = "assumed"
)

// Trusted reports whether a value carrying this provenance may be relied on
// for a preset handed to a user.
//
// ProvAssumed is the only provenance that cannot: a guessed DSP cost cannot
// support a claim that a rig fits, and an over-budget preset that will not
// load is the most visible way this fails.
func (p Provenance) Trusted() bool { return p != ProvAssumed }

// DSPCost is a block's share of one processor, expressed as a fraction of the
// chip. Stereo instances cost more than mono ones.
type DSPCost struct {
	Mono   float64    `json:"mono"`
	Stereo float64    `json:"stereo"`
	Prov   Provenance `json:"prov"`
}

// Param describes one parameter a block accepts.
//
// Min and Max apply to numeric kinds only. A bool has no range, and Line 6
// records its bounds as false and true, which carries no information.
type Param struct {
	Key     string     `json:"key"`
	Label   string     `json:"label"`
	Type    ParamType  `json:"type"`
	Min     float64    `json:"min"`
	Max     float64    `json:"max"`
	Default ParamValue `json:"default"`
	Enum    []string   `json:"enum,omitempty"`
	Unit    string     `json:"unit"`
	Prov    Provenance `json:"prov"`
}

// Block is one model the device can place in a signal chain.
type Block struct {
	ID   ModelID `json:"id"`
	Name string  `json:"name"`
	// Category groups the block by what it does in a chain.
	Category Category `json:"category"`
	// BasedOn is the real-world gear this model emulates, as Line 6 states it
	// in their own documentation — "Ampeg SVT (normal channel)". Empty when
	// the model emulates nothing in particular, such as a utility block.
	//
	// This is what makes a recipe usable. A recipe names gear a person
	// recognises; only this field connects that to a model identifier.
	BasedOn string `json:"based_on,omitempty"`
	// Subcategory is Line 6's own grouping — "Guitar", "Bass". It decides
	// which half of the catalog a request is allowed to draw from.
	Subcategory string `json:"subcategory,omitempty"`
	// CabLink is the cabinet Line 6 pairs with this amp by default. Empty for
	// anything that is not an amp.
	//
	// A recipe that names no cabinet gets this one, which is a better answer
	// than picking arbitrarily: it is the pairing the model was voiced with.
	CabLink ModelID          `json:"cablink,omitempty"`
	Params  map[string]Param `json:"params"`
	Stereo  bool             `json:"stereo"`
	DSP     DSPCost          `json:"dsp"`
	Prov    Provenance       `json:"prov"`
}

// Catalog is every block one device supports, as of one release of the
// software it was generated from.
type Catalog struct {
	Device        string `json:"device"`
	DeviceID      int    `json:"device_id"`
	SchemaVersion int    `json:"schema_version"`
	// Source names the release this was generated from, such as
	// "HX Edit 3.82".
	//
	// A catalog is only true of the models that release knew about. A device
	// running older firmware may not have all of them, and a newer release
	// may add more, so a catalog that cannot say where it came from cannot be
	// checked against anything.
	Source string            `json:"source"`
	Blocks map[ModelID]Block `json:"blocks"`
}

// ParamType names the kind a ParamValue holds.
type ParamType string

// The parameter kinds a Helix block accepts.
const (
	ParamFloat ParamType = "float"
	ParamInt   ParamType = "int"
	ParamBool  ParamType = "bool"
	ParamEnum  ParamType = "enum"
)

// ParamValue is one parameter value of a known kind. Its zero value carries no
// kind and is invalid: marshalling it reports ErrBadParam rather than
// silently emitting a value the device would reject.
type ParamValue struct {
	typ ParamType
	f   float64
	i   int64
	b   bool
	s   string
}
