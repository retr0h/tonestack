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

// Package corpus holds what a body of real presets says about how a device is
// actually used.
//
// The catalog says what a device *can* do; this says what people *do* with it.
// They answer different questions and neither substitutes for the other. Line
// 6 states a default Treble of 0.68 for an Ampeg SVT; across every SVT in the
// corpus the median is 0.845. Both are facts, and the second is the one worth
// generating from.
//
// Nothing here is authority. It is a measurement over presets strangers made,
// including their mistakes — which is why it reports a spread alongside every
// median, and why a wide spread should defer to a person.
package corpus

import "github.com/retr0h/tonestack/pkg/catalog"

// Stats is a whole corpus, measured.
type Stats struct {
	// Device names the hardware these presets were written for.
	Device   string `json:"device"`
	DeviceID int    `json:"device_id"`
	// Presets is how many were measured, so a reader can judge the weight of
	// everything below.
	Presets int `json:"presets"`
	// Models is keyed by the device's own model identifier.
	Models map[catalog.ModelID]ModelStats `json:"models"`
	// Grammar is keyed by instrument — "guitar", "bass" — because a bass
	// chain and a lead chain are built differently and averaging them
	// describes neither.
	Grammar map[string]Grammar `json:"grammar"`
}

// ModelStats is how often one model is used, and how it is set when it is.
type ModelStats struct {
	Uses   int                   `json:"uses"`
	Params map[string]ParamStats `json:"params"`
}

// ParamStats is the distribution of one parameter's values across the corpus.
//
// The quartiles matter as much as the median. A parameter everybody sets the
// same way is one this system can be confident about; a parameter nobody
// agrees on is a judgement call that belongs to the player, and the spread is
// how those are told apart.
type ParamStats struct {
	N      int     `json:"n"`
	Median float64 `json:"median"`
	P25    float64 `json:"p25"`
	P75    float64 `json:"p75"`
}

// Spread is the interquartile range: how much players disagree.
//
// Near zero means consensus and a value worth adopting. Wide means taste, and
// a generated preset should either leave it at the catalog's default or take
// direction from the recipe rather than pretend the median means something.
func (p ParamStats) Spread() float64 { return p.P75 - p.P25 }

// Grammar is what a chain for one instrument tends to contain.
type Grammar struct {
	// Chains is how many were measured.
	Chains int `json:"chains"`
	// Categories is keyed by what a block does.
	Categories map[catalog.Category]CategoryStats `json:"categories"`
}

// CategoryStats is how often a kind of block appears, and where it sits.
type CategoryStats struct {
	// Chains is how many chains hold at least one.
	Chains int `json:"chains"`
	// Before and After count instances either side of the amp. Position in a
	// chain is not decoration: drive before an amp overdrives its input,
	// drive after it does something else entirely.
	Before int `json:"before"`
	After  int `json:"after"`
}

// Frequency is the share of chains holding at least one of these.
func (c CategoryStats) Frequency(chains int) float64 {
	if chains == 0 {
		return 0
	}

	return float64(c.Chains) / float64(chains)
}

// BeforeAmp is the share of instances that sit ahead of the amp.
func (c CategoryStats) BeforeAmp() float64 {
	total := c.Before + c.After
	if total == 0 {
		return 0
	}

	return float64(c.Before) / float64(total)
}
