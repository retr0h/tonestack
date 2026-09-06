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
// Package recipe holds curated knowledge about how a sound is built: which
// gear a player or style uses, and how it should behave.
//
// A recipe names real-world gear — "Ampeg SVT" — and never a device model
// identifier. Resolving one to the other is the catalog's job, which keeps a
// recipe readable, correct when Line 6 renames a model, and usable on any
// device.
//
// The contract is schemas/recipe.schema.json.
package recipe

// Kind distinguishes a named player from a style.
type Kind string

// The kinds of subject a recipe can describe.
const (
	KindArtist Kind = "artist"
	KindGenre  Kind = "genre"
)

// Instrument selects which half of a device catalog a request may draw from.
// Line 6 tags every amp and cab Guitar or Bass.
type Instrument string

// The instruments a recipe can describe.
const (
	InstrumentGuitar Instrument = "guitar"
	InstrumentBass   Instrument = "bass"
)

// Source records where a recipe's knowledge came from.
type Source string

// Where a recipe's knowledge came from, least trustworthy first.
const (
	// SourceLLM was asserted by a language model and nobody checked. Reliable
	// for well-known players, unreliable for obscure ones, and the model
	// cannot always tell which it is doing.
	SourceLLM Source = "llm"
	// SourceCurated was confirmed by a person.
	SourceCurated Source = "curated"
	// SourceCited is traceable to a named source.
	SourceCited Source = "cited"
)

// Confidence is how far a recipe's claims should be trusted.
type Confidence string

// How far a recipe's claims should be trusted.
const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

// Recipe is curated knowledge about one player or style.
type Recipe struct {
	ID             string     `json:"id"`
	Kind           Kind       `json:"kind"`
	Name           string     `json:"name"`
	Band           string     `json:"band,omitempty"`
	InstrumentType Instrument `json:"instrument_type"`
	Aliases        []string   `json:"aliases,omitempty"`
	Rig            Rig        `json:"rig"`
	Character      []string   `json:"character,omitempty"`
	Variants       []Variant  `json:"variants,omitempty"`
	Provenance     Provenance `json:"provenance"`
}

// Rig is the stable layer: career-long for a player, definitional for a style.
//
// A wrong amp here is a bad miss that nothing downstream recovers from. A
// wrong value in Character is a near miss one correction fixes.
type Rig struct {
	Instrument string   `json:"instrument,omitempty"`
	Amp        string   `json:"amp"`
	Cab        string   `json:"cab,omitempty"`
	Pedals     []string `json:"pedals,omitempty"`
	Technique  string   `json:"technique,omitempty"`
}

// Variant is a per-song or per-era departure. Settings move; gear does not.
type Variant struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Character  []string    `json:"character,omitempty"`
	Technique  string      `json:"technique,omitempty"`
	Provenance *Provenance `json:"provenance,omitempty"`
}

// Provenance says where a recipe's knowledge came from and how far to trust it.
type Provenance struct {
	Source     Source     `json:"source"`
	Confidence Confidence `json:"confidence"`
	URL        string     `json:"url,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

// Trusted reports whether a recipe's gear identification was confirmed by a
// person rather than asserted by a model.
func (p Provenance) Trusted() bool {
	return p.Source == SourceCurated || p.Source == SourceCited
}
