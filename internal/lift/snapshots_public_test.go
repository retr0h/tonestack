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

package lift_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// SnapshotsPublicTestSuite covers what a footswitch recalls.
type SnapshotsPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *SnapshotsPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// presetWith returns a document holding the given tone entries.
func (s *SnapshotsPublicTestSuite) presetWith(entries map[string]string) *preset.Document {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	for key := range doc.Data.Tone {
		if key != "dsp0" && key != "dsp1" {
			delete(doc.Data.Tone, key)
		}
	}

	for key, body := range entries {
		var entry preset.Tone
		s.Require().NoError(json.Unmarshal([]byte(body), &entry))

		doc.Data.Tone[key] = entry
	}

	doc.Data.Tone["dsp0"]["block0"] = json.RawMessage(
		`{"@model": "HD2_AmpSVBeastNrm", "@position": 0, "@enabled": true}`)

	return doc
}

// TestLiftSnapshots reads what a footswitch recalls.
func (s *SnapshotsPublicTestSuite) TestLiftSnapshots() {
	tests := []struct {
		name    string
		entries map[string]string
		// the names the snapshots must carry, in order.
		want []string
		// a snapshot naming nothing, with a tempo of its own.
		unnamed bool
		tempo   float64
		// no snapshots at all.
		none bool
	}{
		{
			// Not the order a map happens to iterate: snapshot10 comes after
			// snapshot2, and reading them by name would put it second.
			name: "snapshots in the order they are numbered",
			entries: map[string]string{
				"snapshot2":  `{"@name": "Third"}`,
				"snapshot10": `{"@name": "Eleventh"}`,
				"snapshot0":  `{"@name": "First"}`,
			},
			want: []string{"First", "Third", "Eleventh"},
		},
		{
			name:    "an entry nothing can number",
			entries: map[string]string{"snapshotX": `{"@name": "Nope"}`},
			none:    true,
		},
		{
			// A rig somebody edited can put anything here. A field that
			// cannot be read is left unset rather than written as a zero,
			// which would claim the device said something it did not.
			name:    "a field that will not read",
			entries: map[string]string{"snapshot0": `{"@name": 7, "@tempo": 120}`},
			unnamed: true,
			tempo:   120,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			spec, err := lift.Lift(s.presetWith(tt.entries), s.cat)
			s.Require().NoError(err)

			if tt.none {
				s.Require().Nil(spec.Snapshots)

				return
			}

			if tt.unnamed {
				snap := (*spec.Snapshots)[0]

				s.Require().Nil(snap.Name)
				s.Require().InDelta(tt.tempo, *snap.Tempo, 0.001)

				return
			}

			s.Require().Len(*spec.Snapshots, len(tt.want))

			for i, want := range tt.want {
				s.Require().Equal(want, *(*spec.Snapshots)[i].Name)
			}
		})
	}
}

// TestLowerSnapshots writes a rig's snapshots over the preset's own.
//
// An untouched preset carries three of its own, and keeping those beside a
// rig's would rebuild a preset holding snapshots nobody made.
func (s *SnapshotsPublicTestSuite) TestLowerSnapshots() {
	name := "Verse"

	tests := []struct {
		name string
		// a rig lifted off a preset holding this one snapshot, or one
		// somebody typed.
		lifted string
		typed  *riggen.RigSpec
		want   string
	}{
		{
			name:   "a rig lifted off a preset",
			lifted: "Only",
			want:   `"@name": "Only"`,
		},
		{
			// A rig somebody typed carries no record of a device, so nothing
			// has already cleared the preset it is built into. Its snapshots
			// still have to replace the three an untouched preset ships with.
			name: "a rig somebody typed",
			typed: &riggen.RigSpec{
				Schema:     riggen.RigSpecSchemaRigSpec,
				ID:         "typed",
				Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Typed"},
				Instrument: riggen.InstrumentBass,
				Chain:      []riggen.ChainEntry{{Role: riggen.RoleAmp, Gear: "Ampeg SVT"}},
				Snapshots:  &[]riggen.Snapshot{{Name: &name}},
			},
			want: `"@name": "Verse"`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var spec riggen.RigSpec

			if tt.typed != nil {
				spec = *tt.typed
			} else {
				got, err := lift.Lift(s.presetWith(map[string]string{
					"snapshot0": `{"@name": "` + tt.lifted + `"}`,
				}), s.cat)
				s.Require().NoError(err)

				spec = got
			}

			doc, err := preset.Blank()
			s.Require().NoError(err)
			s.Require().NoError(lift.Lower(doc, spec, s.cat))

			var out bytes.Buffer
			s.Require().NoError(preset.Write(&out, doc))

			s.Require().Contains(out.String(), tt.want)
			s.Require().NotContains(out.String(), "SNAPSHOT 2",
				"the preset's own snapshots are not kept beside the rig's")
		})
	}
}

func TestSnapshotsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SnapshotsPublicTestSuite))
}
