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
package compile_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// SectionsPublicTestSuite covers song sections becoming snapshots.
type SectionsPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *SectionsPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// sectionChain is a compressor, a drive that starts bypassed, and an amp.
var sectionChain = []chain.Block{
	{Model: "HD2_CompressorDeluxeComp", Pos: 0, Enabled: true},
	{Model: "HD2_DM4BuzzSaw", Pos: 1, Enabled: false},
	{Model: "HD2_AmpSVBeastNrm", Pos: 2, Enabled: true},
}

// roles returns a list of roles, for a section to name.
func roles(
	names ...rig.Role,
) *[]rig.Role {
	return &names
}

// snapshot reads one of a document's snapshots back as plain values.
func (s *SectionsPublicTestSuite) snapshot(
	doc *preset.Document,
	key string,
) (string, map[string]map[string]bool) {
	entry := doc.Data.Tone[key]

	var name string
	s.Require().NoError(json.Unmarshal(entry["@name"], &name))

	var blocks map[string]map[string]bool
	if raw, ok := entry["blocks"]; ok {
		s.Require().NoError(json.Unmarshal(raw, &blocks))
	}

	return name, blocks
}

// TestSections writes what each part of a song plays.
func (s *SectionsPublicTestSuite) TestSections() {
	tests := []struct {
		name     string
		sections *[]rig.Section
		snaps    *[]rig.Snapshot
		blocks   []chain.Block
		existing string
		want     map[string]map[string]map[string]bool
		names    map[string]string
		err      error
		errText  string
	}{
		{
			name:   "a rig without sections",
			blocks: sectionChain,
			names:  map[string]string{"snapshot0": "SNAPSHOT 1"},
		},
		{
			// A verse with the drive off and a chorus with it on. The amp
			// and compressor are named by neither, so keep the chain's state.
			name: "a verse and a chorus",
			sections: &[]rig.Section{
				{Name: "Verse", Bypass: roles("drive", "comp")},
				{Name: "Chorus", Play: roles("drive")},
			},
			blocks: sectionChain,
			names: map[string]string{
				"snapshot0": "Verse", "snapshot1": "Chorus", "snapshot2": "SNAPSHOT 3",
			},
			want: map[string]map[string]map[string]bool{
				"snapshot0": {"dsp0": {"block0": false, "block1": false, "block2": true}},
				"snapshot1": {"dsp0": {"block0": true, "block1": true, "block2": true}},
			},
		},
		{
			// What a path records beside its blocks stays, and a block the
			// chain no longer holds goes.
			name:     "written over a snapshot that was already set up",
			sections: &[]rig.Section{{Name: "Solo", Play: roles("drive")}},
			blocks:   sectionChain,
			existing: `{"dsp0": {"block7": true, "blockish": false, "split": true}}`,
			names:    map[string]string{"snapshot0": "Solo"},
			want: map[string]map[string]map[string]bool{
				"snapshot0": {"dsp0": {
					"block0": true, "block1": true, "block2": true,
					"split": true, "blockish": false,
				}},
			},
		},
		{
			name: "more sections than the device has snapshots",
			sections: &[]rig.Section{
				{Name: "Intro"}, {Name: "Verse"}, {Name: "Chorus"}, {Name: "Outro"},
			},
			blocks: sectionChain,
			err:    compile.ErrTooManySections,
		},
		{
			name:     "sections beside snapshots",
			sections: &[]rig.Section{{Name: "Verse"}},
			snaps:    &[]rig.Snapshot{},
			blocks:   sectionChain,
			err:      compile.ErrSectionsAndSnapshots,
		},
		{
			// The chain's roles are listed, and a model the catalog does not
			// carry has none to list.
			name:     "a role the chain has no block for",
			sections: &[]rig.Section{{Name: "Chorus", Bypass: roles("delay")}},
			blocks: append([]chain.Block{{Model: "HD2_FromNewerFirmware", Pos: 3}},
				sectionChain...),
			err:     compile.ErrNoSuchValue,
			errText: "sections[0].bypass: this device has no \"delay\"\n  it has: amp, comp, drive",
		},
		{
			name: "a section that plays and bypasses the same role",
			sections: &[]rig.Section{
				{Name: "Chorus", Play: roles("drive", "amp"), Bypass: roles("drive")},
			},
			blocks:  sectionChain,
			err:     compile.ErrSectionContradicts,
			errText: "sections[0]",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := preset.Blank()
			s.Require().NoError(err)

			if tt.existing != "" {
				doc.Data.Tone["snapshot0"]["blocks"] = json.RawMessage(tt.existing)
			}

			spec := rig.Spec{Sections: tt.sections, Snapshots: tt.snaps}

			err = compile.Sections(doc, spec, tt.blocks, s.cat)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)
				s.Require().Contains(err.Error(), tt.errText)

				name, _ := s.snapshot(doc, "snapshot0")
				s.Require().Equal("SNAPSHOT 1", name, "a refused rig leaves the preset alone")

				return
			}

			s.Require().NoError(err)

			for key, want := range tt.names {
				name, _ := s.snapshot(doc, key)
				s.Require().Equal(want, name)
			}

			for key, want := range tt.want {
				_, blocks := s.snapshot(doc, key)
				s.Require().Equal(want, blocks)
			}
		})
	}
}

// TestLowerRefusesASectionItCannotBuild reaches sections through Lower.
func (s *SectionsPublicTestSuite) TestLowerRefusesASectionItCannotBuild() {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	spec := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "sections",
		Subject:    rig.Subject{Kind: rig.KindSound, Name: "Sections"},
		Instrument: rig.InstrumentBass,
		Chain:      []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT"}},
		Sections:   &[]rig.Section{{Name: "Chorus", Play: roles("drive")}},
	}

	s.Require().ErrorIs(compile.Lower(doc, spec, s.cat), compile.ErrNoSuchValue)
}

func TestSectionsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SectionsPublicTestSuite))
}
