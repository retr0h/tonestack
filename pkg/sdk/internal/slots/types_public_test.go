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

package slots_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/device"
	"github.com/retr0h/tonestack/pkg/sdk/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
	slotmocks "github.com/retr0h/tonestack/pkg/sdk/internal/slots/mocks"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// TypesPublicTestSuite covers standing something else in for a collaborator.
//
// Every other suite here leaves Deps zero and gets the real thing, which is
// what the commands do in earnest. These drive the other half: a command
// given a double uses it, and the failure it reports is the double's.
type TypesPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *TypesPublicTestSuite) SetupTest() { s.ctrl = gomock.NewController(s.T()) }

func (s *TypesPublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// preset returns a standalone .hlx a command can read.
func (s *TypesPublicTestSuite) preset() string {
	return filepath.Join("..", "..", "compile", "testdata", "preset0.hlx")
}

// TestCatalogs covers a command opening its catalog through a double.
func (s *TypesPublicTestSuite) TestCatalogs() {
	want := errors.New("no catalog here")

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Open("somewhere.json").Return(nil, want)

	_, err := slots.Show(slots.ShowOptions{
		Deps:        slots.Deps{Catalogs: cat},
		File:        s.preset(),
		CatalogPath: "somewhere.json",
	})

	s.Require().ErrorIs(err, want)
}

// TestCompiler covers a command reading a preset into a rig through a double.
func (s *TypesPublicTestSuite) TestCompiler() {
	want := errors.New("cannot lift that")

	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Open(gomock.Any()).Return(built, nil)

	comp := slotmocks.NewMockCompiler(s.ctrl)
	comp.EXPECT().Lift(gomock.Any(), gomock.Any()).
		Return(riggen.RigSpec{}, want)

	_, err = slots.Show(slots.ShowOptions{
		Deps: slots.Deps{Catalogs: cat, Compiler: comp},
		File: s.preset(),
	})

	s.Require().ErrorIs(err, want)
}

// TestExportOnARigThatDoesNotValidate covers a lift that produced something
// the contract refuses.
//
// A rig is validated on the way out, so an export that could not write one has
// to say so rather than leave an empty file where somebody expects a preset.
func (s *TypesPublicTestSuite) TestExportOnARigThatDoesNotValidate() {
	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Open(gomock.Any()).Return(built, nil)

	// A rig with no chain in it, which lifting a real preset never produces
	// and the contract does not accept.
	comp := slotmocks.NewMockCompiler(s.ctrl)
	comp.EXPECT().Lift(gomock.Any(), gomock.Any()).
		Return(riggen.RigSpec{}, nil)

	out := filepath.Join(s.T().TempDir(), "rig.yaml")

	_, err = slots.Export(slots.ExportOptions{
		Deps:       slots.Deps{Catalogs: cat, Compiler: comp},
		Path:       fixture("setlist.hls"),
		OutputPath: out,
	})

	s.Require().ErrorContains(err, "writing the rig")
	s.Require().NoFileExists(out)
}

// TestTranslator covers a listing reading each slot's chain through a double.
func (s *TypesPublicTestSuite) TestTranslator() {
	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	raw, err := os.ReadFile(
		filepath.Join("..", "..", "device", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	dev := mocks.NewMockEditor(s.ctrl)
	dev.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.EXPECT().Presets(gomock.Any(), 0).
		Return([]wire.Preset{{Slot: 0, Name: "Chunky Monkey"}}, nil)
	dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(raw, nil)

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Open(gomock.Any()).Return(built, nil)

	// The slot really holds six blocks. The double says it holds none, so a
	// listing that shows it empty can only have asked the double.
	tr := slotmocks.NewMockTranslator(s.ctrl)
	tr.EXPECT().Chain("", gomock.Any(), gomock.Any()).Return(chain.Chain{}, nil)

	listing, err := slots.ListWith(context.Background(), dev,
		slots.DeviceOptions{
			All:  true,
			Deps: slots.Deps{Catalogs: cat, Translator: tr},
		})
	s.Require().NoError(err)

	s.Require().Len(listing.Slots, 1)
	s.Require().Equal("Chunky Monkey", listing.Slots[0].Name)
	s.Require().True(listing.Slots[0].Empty(),
		"the double said the slot holds nothing, and it really holds six")
}

func TestTypesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TypesPublicTestSuite))
}
