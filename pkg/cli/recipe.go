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

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Recipes prints every rig a directory holds, one to a row.
func Recipes(w io.Writer, r sdk.Recipes) error {
	rows := make([][]string, 0, len(r.Rigs))

	for _, spec := range r.Rigs {
		rows = append(rows, []string{
			paint.Accent(w, spec.ID),
			spec.Subject.Name,
			paint.Mute(w, string(spec.Instrument)),
			rig.GearName(spec, rig.RoleAmp),
			source(w, spec),
		})
	}

	return wrapReport(paint.Section{
		Title:   "Recipes",
		Detail:  r.Dir,
		Headers: []string{"id", "name", "instrument", "amp", "source"},
		Rows:    rows,
		Empty:   "no recipes here",
	}.Render(w))
}

// Recipe prints one rig in full.
func Recipe(w io.Writer, r sdk.Recipe) error {
	spec := r.Rig
	d := paint.Detail{Title: spec.Subject.Name, Subtitle: spec.ID}

	if spec.Subject.Band != nil && *spec.Subject.Band != "" {
		d.Fields = append(d.Fields, paint.Field{Label: "band", Value: *spec.Subject.Band})
	}

	if spec.Subject.Era != nil && *spec.Subject.Era != "" {
		d.Fields = append(d.Fields, paint.Field{Label: "era", Value: *spec.Subject.Era})
	}

	d.Fields = append(d.Fields,
		paint.Field{Label: "instrument", Value: string(spec.Instrument)})
	d.Fields = append(d.Fields, signalPath(spec)...)

	if spec.Technique != nil {
		d.Fields = append(d.Fields,
			paint.Field{Label: "technique", Value: technique(*spec.Technique)})
	}

	d.Fields = append(d.Fields, character(spec)...)
	d.Fields = append(d.Fields, variants(r.Variants)...)
	d.Fields = append(d.Fields, paint.Field{
		Label: "source",
		Value: fmt.Sprintf("%s, %s confidence",
			rig.Sourced(spec), confidence(spec)),
	})

	if !rig.Trusted(spec) {
		d.Note = "unverified — nobody has confirmed this gear"
	}

	return wrapReport(d.Render(w))
}

// variants lists the rigs that are a small change on this one.
//
// Only the first carries the label, so several read as one block rather than
// as the same word repeated down the page.
func variants(all []sdk.Variant) []paint.Field {
	out := make([]paint.Field, 0, len(all))

	for _, v := range all {
		label := ""
		if len(out) == 0 {
			label = "variants"
		}

		out = append(out, paint.Field{
			Label: label,
			Value: fmt.Sprintf("%s (%s)", v.Name, v.ID),
		})
	}

	return out
}

// source names where a rig's knowledge came from, and marks it when nobody
// has confirmed it.
//
// This is the column that decides whether to trust the row, so it is the one
// that carries colour. A rig is only shown as confirmed when every claim in
// it rests on something checkable — the weakest link is what the reader needs
// to know about.
func source(w io.Writer, spec rig.Spec) string {
	if rig.Trusted(spec) {
		return paint.OK(w, string(rig.Sourced(spec)))
	}

	return paint.Info(w, string(rig.Sourced(spec)))
}

// chain renders the signal path, in order, one row per piece of gear.
//
// Labelled by role rather than by position, because "amp" is what a person
// reading this wants to find and "3" is not.
func signalPath(spec rig.Spec) []paint.Field {
	out := make([]paint.Field, 0, len(spec.Chain))

	for _, e := range spec.Chain {
		out = append(out, paint.Field{Label: string(e.Role), Value: e.Gear})
	}

	return out
}

// confidence reports how far a rig says it should be trusted.
func confidence(spec rig.Spec) rig.Confidence {
	if spec.Confidence == nil {
		return rig.ConfidenceLow
	}

	return *spec.Confidence
}

// character renders the intent lines, one per row, labelled only once.
//
// The label repeats as blank so the values line up in the same column as
// every other field rather than starting a block of their own.
func character(spec rig.Spec) []paint.Field {
	if spec.Character == nil || len(*spec.Character) == 0 {
		return nil
	}

	out := make([]paint.Field, 0, len(*spec.Character))

	for i, c := range *spec.Character {
		label := ""
		if i == 0 {
			label = "character"
		}

		out = append(out, paint.Field{Label: label, Value: c.Term})
	}

	return out
}

// where reads a position back as the phrase a player would use.
var where = map[rig.Position]string{
	rig.PositionBridge: "near the bridge",
	rig.PositionMiddle: "over the middle",
	rig.PositionNeck:   "over the neck",
}

// technique writes the three things a rig stores as the one sentence a person
// would say.
//
// A rig stores them apart so that two rigs can be compared, and nobody says
// "attack: pick, position: bridge" out loud. Muting is named only when there
// is some, because "not muted" is what every unmuted note already sounds like.
func technique(t rig.Technique) string {
	parts := []string{string(t.Attack)}

	if t.Position != nil {
		parts = append(parts, where[*t.Position])
	}

	if t.Muting != nil && *t.Muting == rig.MutingPalm {
		parts = append(parts, "palm muted")
	}

	return strings.Join(parts, ", ")
}

// wrapReport gives a reporting failure the same shape everywhere.
func wrapReport(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("reporting: %w", err)
}

// Scaffolded says what recipe was written and what to do with it.
func Scaffolded(w io.Writer, sc sdk.Scaffolded) error {
	rows := [][]string{
		{paint.Mute(w, "id"), paint.Accent(w, sc.ID)},
		{paint.Mute(w, "instrument"), sc.Instrument},
		{paint.Mute(w, "amp"), sc.Amp},
	}

	if sc.Cab != "" {
		rows = append(rows, []string{paint.Mute(w, "cab"), sc.Cab})
	}

	if len(sc.Pedals) > 0 {
		rows = append(rows,
			[]string{paint.Mute(w, "pedals"), strings.Join(sc.Pedals, ", ")})
	}

	if err := (paint.Section{
		Title: sc.Name, Detail: sc.Path, Rows: rows,
		Summary: fmt.Sprintf(
			"every gear name resolves — next: tonestack presets make --id %s --out %s.hlx",
			sc.ID, sc.ID),
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}
