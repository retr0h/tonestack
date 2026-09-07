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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/rig"
	"github.com/retr0h/tonestack/pkg/rig/gen"
)

type ReadPublicTestSuite struct {
	suite.Suite
}

// spec returns a rig holding the given chain, with nothing else filled in.
func spec(chain ...gen.ChainEntry) gen.RigSpec {
	return gen.RigSpec{
		Schema:     gen.RigSpecSchemaRigSpec,
		ID:         "test",
		Subject:    gen.Subject{Kind: gen.KindArtist, Name: "Test"},
		Instrument: gen.InstrumentBass,
		Chain:      chain,
	}
}

// evidence returns the kinds as a rig carries them.
func evidence(kinds ...gen.EvidenceKind) *[]gen.Evidence {
	out := make([]gen.Evidence, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, gen.Evidence{Kind: k})
	}

	return &out
}

func (s *ReadPublicTestSuite) TestGearFindsARole() {
	got, ok := rig.Gear(spec(
		gen.ChainEntry{Role: gen.RoleDrive, Gear: "Klon Centaur"},
		gen.ChainEntry{Role: gen.RoleAmp, Gear: "Ampeg SVT"},
	), gen.RoleAmp)

	s.Require().True(ok)
	s.Require().Equal("Ampeg SVT", got.Gear)
}

func (s *ReadPublicTestSuite) TestGearReportsARoleTheChainDoesNotHave() {
	_, ok := rig.Gear(spec(gen.ChainEntry{Role: gen.RoleAmp}), gen.RoleCab)

	s.Require().False(ok)
}

func (s *ReadPublicTestSuite) TestGearNameIsEmptyForARoleTheChainLacks() {
	s.Require().Empty(rig.GearName(spec(), gen.RoleAmp))
}

func (s *ReadPublicTestSuite) TestGearNameNamesTheGear() {
	s.Require().Equal("Ampeg SVT", rig.GearName(
		spec(gen.ChainEntry{Role: gen.RoleAmp, Gear: "Ampeg SVT"}), gen.RoleAmp))
}

func (s *ReadPublicTestSuite) TestAClaimNobodySupportedIsNotTrusted() {
	s.Require().False(rig.Trusted(spec(gen.ChainEntry{Role: gen.RoleAmp})))
}

func (s *ReadPublicTestSuite) TestAnAssertionAloneIsNotTrusted() {
	// `llm` means a model said so and nobody checked, which is the same
	// standing as nobody having said anything.
	s.Require().False(rig.Trusted(spec(gen.ChainEntry{
		Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceLLM),
	})))
}

func (s *ReadPublicTestSuite) TestOneUnsupportedClaimIsEnoughToDistrustARig() {
	s.Require().False(rig.Trusted(spec(
		gen.ChainEntry{Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceCited)},
		gen.ChainEntry{Role: gen.RoleCab, Evidence: evidence(gen.EvidenceLLM)},
	)))
}

func (s *ReadPublicTestSuite) TestEveryClaimSupportedIsTrusted() {
	s.Require().True(rig.Trusted(spec(
		gen.ChainEntry{Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceCited)},
		gen.ChainEntry{Role: gen.RoleCab, Evidence: evidence(gen.EvidenceVideo)},
	)))
}

func (s *ReadPublicTestSuite) TestEvidenceOnTheRigCoversTheWholeChain() {
	// A rig rundown covers every piece of gear in it. Requiring the citation
	// on each entry would only encourage repeating it.
	r := spec(gen.ChainEntry{Role: gen.RoleAmp})
	r.Evidence = evidence(gen.EvidenceCited)

	s.Require().True(rig.Trusted(r))
}

func (s *ReadPublicTestSuite) TestARigWithNoChainIsNotTrusted() {
	s.Require().False(rig.Trusted(spec()))
}

func (s *ReadPublicTestSuite) TestSourcedNamesTheStrongestEvidence() {
	r := spec(
		gen.ChainEntry{Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceLLM)},
		gen.ChainEntry{Role: gen.RoleCab, Evidence: evidence(gen.EvidenceCited)},
	)
	r.Evidence = evidence(gen.EvidenceCorpus)

	s.Require().Equal(gen.EvidenceCited, rig.Sourced(r))
}

func (s *ReadPublicTestSuite) TestSourcedRanksAPersonAboveEverything() {
	// Nothing in this project can hear, so somebody who listened outranks any
	// citation.
	s.Require().Equal(gen.EvidenceUser, rig.Sourced(spec(gen.ChainEntry{
		Role: gen.RoleAmp,
		Evidence: evidence(
			gen.EvidenceMeasured, gen.EvidenceUser, gen.EvidenceCited),
	})))
}

func (s *ReadPublicTestSuite) TestSourcedRanksEveryKind() {
	for _, tc := range []struct {
		name  string
		kinds []gen.EvidenceKind
		want  gen.EvidenceKind
	}{
		{"measured over cited", []gen.EvidenceKind{
			gen.EvidenceCited, gen.EvidenceMeasured,
		}, gen.EvidenceMeasured},
		{"cited over video", []gen.EvidenceKind{
			gen.EvidenceVideo, gen.EvidenceCited,
		}, gen.EvidenceCited},
		{"video over audio", []gen.EvidenceKind{
			gen.EvidenceAudio, gen.EvidenceVideo,
		}, gen.EvidenceVideo},
		{"audio over corpus", []gen.EvidenceKind{
			gen.EvidenceCorpus, gen.EvidenceAudio,
		}, gen.EvidenceAudio},
		{"corpus over an assertion", []gen.EvidenceKind{
			gen.EvidenceLLM, gen.EvidenceCorpus,
		}, gen.EvidenceCorpus},
		{"a kind nobody has ranked", []gen.EvidenceKind{
			gen.EvidenceKind("seance"),
		}, gen.EvidenceKind("seance")},
		{"an unranked kind does not outrank a known one", []gen.EvidenceKind{
			gen.EvidenceCorpus, gen.EvidenceKind("seance"),
		}, gen.EvidenceCorpus},
	} {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, rig.Sourced(spec(gen.ChainEntry{
				Role: gen.RoleAmp, Evidence: evidence(tc.kinds...),
			})))
		})
	}
}

func (s *ReadPublicTestSuite) TestSourcedOnARigNobodySupported() {
	// Nothing said where it came from, which is the same standing as a model
	// having asserted it.
	s.Require().Equal(gen.EvidenceLLM, rig.Sourced(spec(
		gen.ChainEntry{Role: gen.RoleAmp},
	)))
}

func TestReadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadPublicTestSuite))
}
