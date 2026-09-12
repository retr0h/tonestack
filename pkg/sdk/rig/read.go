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

package rig

// gear returns the first entry filling a role, and whether the rig has one.
//
// A chain is ordered by what the signal does rather than grouped by kind, so
// finding the amplifier means looking for it. Every caller that displays a rig
// wants this and none of them should search the chain themselves.
func gear(spec Spec, role Role) (ChainEntry, bool) {
	for _, e := range spec.Chain {
		if e.Role == role {
			return e, true
		}
	}

	return ChainEntry{}, false
}

// GearName returns the gear filling a role, or an empty string.
func GearName(spec Spec, role Role) string {
	if e, ok := gear(spec, role); ok {
		return e.Gear
	}

	return ""
}

// Trusted reports whether every claim in a rig rests on something checkable.
//
// A rig states its own confidence, and a rig may state high confidence with
// nothing behind it. What matters is whether somebody could go and look: a
// citation, a video, a measurement, or a person who listened. `llm` alone
// means a model asserted it and nobody checked, which is this project's
// largest correctness risk and is worth showing rather than leaving implied.
//
// Evidence attaches to a claim, so each piece of gear answers for itself.
// Evidence on the rig answers for all of it — a rig rundown covers every
// piece of gear in it, and requiring the citation to be repeated on each
// entry would only encourage repeating it.
func Trusted(spec Spec) bool {
	if checkable(spec.Evidence) {
		return true
	}

	if len(spec.Chain) == 0 {
		return false
	}

	for _, e := range spec.Chain {
		if !checkable(e.Evidence) {
			return false
		}
	}

	return true
}

// checkable reports whether evidence holds anything but an assertion.
//
// Absent evidence is not checkable either. A claim nobody supported and a
// claim a model asserted are the same claim.
func checkable(evidence *[]Evidence) bool {
	if evidence == nil {
		return false
	}

	for _, e := range *evidence {
		if e.Kind != EvidenceLLM {
			return true
		}
	}

	return false
}

// Sourced names where a rig's knowledge came from, strongest first.
//
// Ranked by how far somebody has to go to disagree with it. A person who
// listened outranks a citation, because this project's founding constraint is
// that nothing in it can hear.
func Sourced(spec Spec) EvidenceKind {
	best := EvidenceKind("")
	rank := func(k EvidenceKind) int {
		switch k {
		case EvidenceUser:
			return 6
		case EvidenceMeasured:
			return 5
		case EvidenceCited:
			return 4
		case EvidenceVideo:
			return 3
		case EvidenceAudio:
			return 2
		case EvidenceCorpus:
			return 1
		case EvidenceLLM:
			return 0
		default:
			return 0
		}
	}

	each := func(evidence *[]Evidence) {
		if evidence == nil {
			return
		}

		for _, e := range *evidence {
			if best == "" || rank(e.Kind) > rank(best) {
				best = e.Kind
			}
		}
	}

	each(spec.Evidence)

	for _, e := range spec.Chain {
		each(e.Evidence)
	}

	if best == "" {
		return EvidenceLLM
	}

	return best
}
