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

package editor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// RoutingStatesTestSuite covers reading a preset's routing for a device.
//
// A device sends fewer values than a model has names for, so what matters
// here is that each list comes back the length the device writes and in the
// order the catalog gives, rather than the length the catalog implies.
type RoutingStatesTestSuite struct {
	suite.Suite

	cat  *catalog.Catalog
	held []wire.DeviceRouting
}

func (s *RoutingStatesTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)

	s.held = wire.BlankRouting()
	s.Require().NotEmpty(s.held)
}

// preset0 is a preset HX Edit wrote, with routing somebody set.
func (s *RoutingStatesTestSuite) preset0() *preset.Document {
	f, err := os.Open(
		filepath.Join("..", "compile", "testdata", "preset0.hlx"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := preset.Read(f)
	s.Require().NoError(err)

	return doc
}

// slot finds one entry in what was read.
func (s *RoutingStatesTestSuite) slot(
	got []wire.Routing,
	name string,
) wire.Routing {
	for _, r := range got {
		if r.Slot == name {
			return r
		}
	}

	s.Require().FailNow("no routing slot " + name)

	return wire.Routing{}
}

// TestAnOutputCarriesWhatTheFileSays is the case the task was filed for.
func (s *RoutingStatesTestSuite) TestAnOutputCarriesWhatTheFileSays() {
	got := RoutingStates(s.preset0(), s.cat, s.held)

	out := s.slot(got, "outputA")

	// Two, not the three the catalog names: a device stores `select` under
	// its own key rather than among the values.
	s.Require().NotNil(out.Values)
	s.Require().Len(*out.Values, 2)
	s.Require().Equal(2, out.Named)

	s.Require().InDelta(0.5, (*out.Values)[0], 0.0001, "pan")
	s.Require().InDelta(-2.9, (*out.Values)[1], 0.0001, "gain")

	s.Require().NotNil(out.Select)
	s.Require().Equal(1, *out.Select)
}

// TestAnInputIsCappedAtWhatADeviceSends covers the widest gap.
//
// The model names seven parameters and a device sends three.
func (s *RoutingStatesTestSuite) TestAnInputIsCappedAtWhatADeviceSends() {
	got := RoutingStates(s.preset0(), s.cat, s.held)

	in := s.slot(got, "inputA")

	s.Require().NotNil(in.Values)
	s.Require().Len(*in.Values, 3, "noiseGate, threshold and decay")

	s.Require().Equal(false, (*in.Values)[0], "noiseGate")
	s.Require().InDelta(-48, (*in.Values)[1], 0.0001, "threshold")
	s.Require().InDelta(0.5, (*in.Values)[2], 0.0001, "decay")
}

// TestASplitCarriesItsModelAndPlace covers the fields only a split has.
func (s *RoutingStatesTestSuite) TestASplitCarriesItsModelAndPlace() {
	got := RoutingStates(s.preset0(), s.cat, s.held)

	split := s.slot(got, "split")

	s.Require().NotNil(split.Model)
	s.Require().NotNil(split.Enabled)
	s.Require().True(*split.Enabled)
	s.Require().NotNil(split.Position)
	s.Require().Equal(0, *split.Position)

	s.Require().NotNil(split.Values)
	s.Require().Len(*split.Values, 3, "BalanceA, BalanceB and bypass")
}

// TestAnInputCarriesNoModelOrPlace covers what a device keeps to itself.
func (s *RoutingStatesTestSuite) TestAnInputCarriesNoModelOrPlace() {
	got := RoutingStates(s.preset0(), s.cat, s.held)

	in := s.slot(got, "inputA")

	s.Require().Nil(in.Model, "a device knows which input is its own")
	s.Require().Nil(in.Enabled)
	s.Require().Nil(in.Position)
}

// TestAPresetWithNoChain carries no routing either.
func (s *RoutingStatesTestSuite) TestAPresetWithNoChain() {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	delete(doc.Data.Tone, processorKey)

	s.Require().Nil(RoutingStates(doc, s.cat, s.held))
}

// TestAPresetNamingNoneOfThem carries nothing to write.
func (s *RoutingStatesTestSuite) TestAPresetNamingNoneOfThem() {
	doc := s.preset0()

	for _, r := range s.held {
		delete(doc.Data.Tone[processorKey], r.Slot)
	}

	s.Require().Nil(RoutingStates(doc, s.cat, s.held))
}

// TestAnEntryThatWillNotRead is skipped rather than guessed at.
func (s *RoutingStatesTestSuite) TestAnEntryThatWillNotRead() {
	doc := s.preset0()
	doc.Data.Tone[processorKey]["outputA"] = json.RawMessage(`nonsense`)

	got := RoutingStates(doc, s.cat, s.held)

	for _, r := range got {
		s.Require().NotEqual("outputA", r.Slot)
	}
}

// TestAnEntryNamingNoModel carries no values.
//
// The model is the only thing that says which parameters an entry has, and a
// list built without one would put values in the wrong places.
func (s *RoutingStatesTestSuite) TestAnEntryNamingNoModel() {
	doc := s.preset0()
	doc.Data.Tone[processorKey]["outputA"] = json.RawMessage(`{"@output": 1}`)

	out := s.slot(RoutingStates(doc, s.cat, s.held), "outputA")

	s.Require().Nil(out.Values)
	s.Require().NotNil(out.Select, "what it could read, it read")
}

// TestAnEntryNamingAModelNobodyCarries also carries no values.
func (s *RoutingStatesTestSuite) TestAnEntryNamingAModelNobodyCarries() {
	doc := s.preset0()
	doc.Data.Tone[processorKey]["outputA"] = json.RawMessage(
		`{"@output": 1, "@model": "HD2_NoSuchFlow"}`)

	out := s.slot(RoutingStates(doc, s.cat, s.held), "outputA")

	s.Require().Nil(out.Values)
}

// TestAModelNameThatWillNotRead is the same as naming none.
func (s *RoutingStatesTestSuite) TestAModelNameThatWillNotRead() {
	doc := s.preset0()
	doc.Data.Tone[processorKey]["outputA"] = json.RawMessage(`{"@model": 7}`)

	out := s.slot(RoutingStates(doc, s.cat, s.held), "outputA")

	s.Require().Nil(out.Values)
}

// TestAParameterThePresetOmits still takes its place.
//
// Position is the only thing naming a value on the wire, so a missing one
// cannot be left out without moving every value after it.
func (s *RoutingStatesTestSuite) TestAParameterThePresetOmits() {
	doc := s.preset0()
	doc.Data.Tone[processorKey]["outputA"] = json.RawMessage(
		`{"@output": 1, "@model": "HelixStomp_AppDSPFlowOutputMain", "gain": -2.9}`)

	out := s.slot(RoutingStates(doc, s.cat, s.held), "outputA")

	s.Require().NotNil(out.Values)
	s.Require().Len(*out.Values, 2)
	s.Require().InDelta(0, (*out.Values)[0], 0.0001, "the pan it does not name")
	s.Require().InDelta(-2.9, (*out.Values)[1], 0.0001)
}

// TestAParameterThatWillNotRead leaves that one at nothing.
func (s *RoutingStatesTestSuite) TestAParameterThatWillNotRead() {
	doc := s.preset0()
	doc.Data.Tone[processorKey]["outputA"] = json.RawMessage(
		`{"@model": "HelixStomp_AppDSPFlowOutputMain", "pan": "loud", "gain": -1}`)

	out := s.slot(RoutingStates(doc, s.cat, s.held), "outputA")

	s.Require().NotNil(out.Values)
	s.Require().InDelta(0, (*out.Values)[0], 0.0001)
	s.Require().InDelta(-1, (*out.Values)[1], 0.0001)
}

func TestRoutingStatesTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RoutingStatesTestSuite))
}
