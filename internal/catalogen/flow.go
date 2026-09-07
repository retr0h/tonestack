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
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// Suffixes Line 6 give the models a device wraps a chain in.
//
// The same shapes are named per device: an HX Stomp writes
// `HelixStomp_AppDSPFlowInput` where a Helix Floor writes
// `HD2_AppDSPFlow1Input`. Both ship in io.models with the devices they belong
// to, so which is which is a question of suffix rather than of guessing.
const (
	flowInput      = "Input"
	flowOutputMain = "OutputMain"
	flowOutputSend = "OutputSend"
	flowOutput     = "Output"
	flowMarker     = "AppDSPFlow"
)

// flowFor picks the models this device puts either side of a chain.
//
// A preset read over USB describes its inputs and outputs without naming
// them, because the device knows which are its own. A file has to name them.
//
// Devices with one output name it `Output`; devices with two name them
// `OutputMain` and `OutputSend`. The first is read as the main pair, which is
// what a preset written for such a device does with it.
func flowFor(models []wireModel, deviceID int) catalog.Flow {
	var out catalog.Flow

	for _, m := range models {
		if !strings.Contains(m.SymbolicID, flowMarker) || !supports(m, deviceID) {
			continue
		}

		// Models belonging to no device in particular are the shared ones:
		// the splits and the join, which a preset names for itself.
		if len(m.Devices) == 0 {
			continue
		}

		switch {
		case strings.HasSuffix(m.SymbolicID, flowOutputMain):
			out.OutputMain = catalog.ModelID(m.SymbolicID)
		case strings.HasSuffix(m.SymbolicID, flowOutputSend):
			out.OutputSend = catalog.ModelID(m.SymbolicID)
		case strings.HasSuffix(m.SymbolicID, flowOutput):
			out.OutputMain = catalog.ModelID(m.SymbolicID)
		case strings.HasSuffix(m.SymbolicID, flowInput):
			// A device with two inputs names them 1 and 2 and uses the first.
			if out.Input == "" || strings.Contains(m.SymbolicID, "Flow1") {
				out.Input = catalog.ModelID(m.SymbolicID)
			}
		}
	}

	return out
}
