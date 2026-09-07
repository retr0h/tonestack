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

package catalogen

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// FlowTestSuite covers picking the models a device wraps a chain in.
//
// A preset read over USB names its blocks and not its inputs, because the
// device knows which are its own. A file has to name them, so this is where
// the names come from.
type FlowTestSuite struct {
	suite.Suite
}

// stomp is the device these fixtures belong to.
const stomp = 2162694

// model returns one entry from io.models.
func model(id string, devices ...int) wireModel {
	out := wireModel{SymbolicID: id}
	for _, d := range devices {
		out.Devices = append(out.Devices, wireDevice{ID: d})
	}

	return out
}

func (s *FlowTestSuite) TestPicksWhatThisDeviceUses() {
	got := flowFor([]wireModel{
		model("HelixStomp_AppDSPFlowInput", stomp, 2162699),
		model("HelixStomp_AppDSPFlowOutputMain", stomp, 2162699),
		model("HelixStomp_AppDSPFlowOutputSend", stomp, 2162699),
		model("HelixFx_AppDSPFlowInput", 2162693),
		model("HelixFx_AppDSPFlowOutput", 2162693),
	}, stomp)

	s.Require().Equal(catalog.ModelID("HelixStomp_AppDSPFlowInput"), got.Input)
	s.Require().Equal(
		catalog.ModelID("HelixStomp_AppDSPFlowOutputMain"), got.OutputMain)
	s.Require().Equal(
		catalog.ModelID("HelixStomp_AppDSPFlowOutputSend"), got.OutputSend)
}

func (s *FlowTestSuite) TestADeviceWithOneOutput() {
	// A device with one output names it `Output` rather than `OutputMain`,
	// and a preset written for it uses that as the main pair.
	got := flowFor([]wireModel{
		model("HD2_AppDSPFlow1Input", 2162689),
		model("HD2_AppDSPFlow2Input", 2162689),
		model("HD2_AppDSPFlowOutput", 2162689),
	}, 2162689)

	s.Require().Equal(catalog.ModelID("HD2_AppDSPFlow1Input"), got.Input)
	s.Require().Equal(catalog.ModelID("HD2_AppDSPFlowOutput"), got.OutputMain)
	s.Require().Empty(got.OutputSend)
}

func (s *FlowTestSuite) TestIgnoresTheSharedShapes() {
	// The splits and the join belong to no device in particular, and a preset
	// names them for itself, so they are not part of this.
	got := flowFor([]wireModel{
		model("HD2_AppDSPFlowSplitY"),
		model("HD2_AppDSPFlowJoin"),
		model("HD2_AmpSVBeastNrm", stomp),
	}, stomp)

	s.Require().Equal(catalog.Flow{}, got)
}

func (s *FlowTestSuite) TestADeviceWithNoneOfThem() {
	s.Require().Equal(catalog.Flow{}, flowFor(nil, stomp))
}

func TestFlowTestSuite(t *testing.T) {
	suite.Run(t, new(FlowTestSuite))
}
