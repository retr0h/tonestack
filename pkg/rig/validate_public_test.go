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

package rig_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/rig"
	"github.com/retr0h/tonestack/pkg/rig/gen"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

// good returns the smallest rig the schema accepts.
func (s *ValidatePublicTestSuite) good() gen.RigSpec {
	return gen.RigSpec{
		Schema:     gen.RigSpecSchemaRigSpec,
		ID:         "mike-dirnt",
		Subject:    gen.Subject{Kind: gen.KindArtist, Name: "Mike Dirnt"},
		Instrument: gen.InstrumentBass,
		Chain:      []gen.ChainEntry{{Role: gen.RoleAmp, Gear: "Ampeg SVT"}},
	}
}

func (s *ValidatePublicTestSuite) TestTheSmallestUsefulRigIsValid() {
	s.Require().NoError(rig.Validate(s.good()))
}

func (s *ValidatePublicTestSuite) TestRejects() {
	settings := func(v float64) *gen.Settings {
		out := gen.Settings{"drive": v}

		return &out
	}
	confidence := func(c gen.Confidence) *gen.Confidence { return &c }

	tests := []struct {
		name   string
		mutate func(*gen.RigSpec)
		field  string
	}{
		{
			"a document that is not a rig",
			func(r *gen.RigSpec) { r.Schema = "L6Preset" },
			"schema",
		},
		{
			"an identifier with spaces",
			func(r *gen.RigSpec) { r.ID = "Mike Dirnt" },
			"id",
		},
		{
			"an identifier that is empty",
			func(r *gen.RigSpec) { r.ID = "" },
			"id",
		},
		{
			"a subject of no known kind",
			func(r *gen.RigSpec) { r.Subject.Kind = "robot" },
			"subject.kind",
		},
		{
			"a subject nobody named",
			func(r *gen.RigSpec) { r.Subject.Name = "  " },
			"subject.name",
		},
		{
			"an instrument the catalog cannot be filtered by",
			func(r *gen.RigSpec) { r.Instrument = "theremin" },
			"instrument",
		},
		{
			"a chain holding nothing",
			func(r *gen.RigSpec) { r.Chain = nil },
			"chain",
		},
		{
			"gear doing nothing in particular",
			func(r *gen.RigSpec) { r.Chain[0].Role = "vibe" },
			"chain[0].role",
		},
		{
			"gear with no name",
			func(r *gen.RigSpec) { r.Chain[0].Gear = "" },
			"chain[0].gear",
		},
		{
			"a setting above one",
			func(r *gen.RigSpec) { r.Chain[0].Settings = settings(1.5) },
			"chain[0].settings.drive",
		},
		{
			"a setting below nought",
			func(r *gen.RigSpec) { r.Chain[0].Settings = settings(-0.1) },
			"chain[0].settings.drive",
		},
		{
			"a confidence nobody can act on",
			func(r *gen.RigSpec) { r.Chain[0].Confidence = confidence("certain") },
			"chain[0].confidence",
		},
		{
			"evidence of no known kind",
			func(r *gen.RigSpec) {
				r.Chain[0].Evidence = &[]gen.Evidence{{Kind: "vibes"}}
			},
			"chain[0].evidence[0].kind",
		},
		{
			"a correction that records no request",
			func(r *gen.RigSpec) {
				r.Mutations = &[]gen.Mutation{{Ask: "  "}}
			},
			"mutations[0].ask",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := s.good()
			tc.mutate(&got)

			err := rig.Validate(got)

			s.Require().ErrorIs(err, rig.ErrInvalid)
			s.Require().Contains(err.Error(), tc.field)
		})
	}
}

func (s *ValidatePublicTestSuite) TestAcceptsEverythingOptional() {
	// Absent is not invalid. A hand-written rig carries almost none of this.
	got := s.good()
	settings := gen.Settings{"drive": 0, "treble": 1}
	got.Chain[0].Settings = &settings
	got.Chain[0].Evidence = &[]gen.Evidence{{Kind: gen.EvidenceCited}}
	got.Mutations = &[]gen.Mutation{{Ask: "make it clunkier"}}

	s.Require().NoError(rig.Validate(got))
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}

func (s *ValidatePublicTestSuite) TestReportsARigItCannotRead() {
	// A rig carries raw JSON it was handed — the state a device wrote — and
	// something that is not JSON cannot be checked against anything.
	spec := s.good()
	broken := json.RawMessage("not json")
	spec.Device = &gen.DeviceState{Version: &broken}

	err := rig.Validate(spec)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reading the rig")
}
