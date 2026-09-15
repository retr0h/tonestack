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
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
	slotmocks "github.com/retr0h/tonestack/pkg/sdk/internal/slots/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// TypesPublicTestSuite covers standing something else in for a collaborator.
//
// Every other suite here leaves the collaborators on Flows nil and gets the
// real thing, which is what the Client does in earnest. These drive the other
// half: a flow given a double uses it, and the failure it reports is the
// double's.
type TypesPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *TypesPublicTestSuite) SetupTest() { s.ctrl = gomock.NewController(s.T()) }

func (s *TypesPublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// preset returns a standalone .hlx a flow can read.
func (s *TypesPublicTestSuite) preset() string {
	return filepath.Join("..", "compile", "testdata", "preset0.hlx")
}

// builtIn is a double handing over the catalog in this binary.
func (s *TypesPublicTestSuite) builtIn() *slotmocks.MockCatalogs {
	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Catalog(gomock.Any()).Return(built, nil)

	return cat
}

// TestCatalogs covers a flow reaching its catalog through a double.
func (s *TypesPublicTestSuite) TestCatalogs() {
	want := errors.New("no catalog here")

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Catalog(gomock.Any()).Return(nil, want)

	_, err := (&slots.Flows{Catalogs: cat}).ShowFile(context.Background(), s.preset())

	s.Require().ErrorIs(err, want)
}

// TestCompiler covers a flow reading a preset into a rig through a double.
func (s *TypesPublicTestSuite) TestCompiler() {
	tests := []struct {
		name string
		// what the lift answers.
		lifted rig.Spec
		err    error
		call   func(*slots.Flows, string) error
		says   string
	}{
		{
			name: "a lift that fails",
			err:  errors.New("cannot lift that"),
			call: func(f *slots.Flows, _ string) error {
				_, err := f.ShowFile(context.Background(), s.preset())

				return err
			},
			says: "cannot lift that",
		},
		{
			// A rig is validated on the way out, so an export that could not
			// write one has to say so rather than leave an empty file where
			// somebody expects a preset. A rig with no chain in it is one
			// lifting a real preset never produces and the contract refuses.
			name: "a lift that produced something the contract refuses",
			call: func(f *slots.Flows, out string) error {
				_, err := f.Export(context.Background(),
					fixture("setlist.hls"), slotpkg.Address{}, out, result.FormatRig)

				return err
			},
			says: "writing the rig",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			comp := slotmocks.NewMockCompiler(s.ctrl)
			comp.EXPECT().Lift(gomock.Any(), gomock.Any()).Return(tt.lifted, tt.err)

			out := filepath.Join(s.T().TempDir(), "rig.yaml")

			err := tt.call(&slots.Flows{Catalogs: s.builtIn(), Compiler: comp}, out)

			s.Require().ErrorContains(err, tt.says)
			s.Require().NoFileExists(out)
		})
	}
}

// TestTranslator covers a listing reading each slot's chain through a double.
func (s *TypesPublicTestSuite) TestTranslator() {
	raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	dev := mocks.NewMockEditor(s.ctrl)
	dev.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.EXPECT().Presets(gomock.Any(), 0).
		Return([]wire.Preset{{Slot: 0, Name: "Chunky Monkey"}}, nil)
	dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(raw, nil)

	// The slot really holds six blocks. The double says it holds none, so a
	// listing that shows it empty can only have asked the double.
	tr := slotmocks.NewMockTranslator(s.ctrl)
	tr.EXPECT().Chain("", gomock.Any(), gomock.Any()).Return(chain.Chain{}, nil)

	listing, err := (&slots.Flows{Catalogs: s.builtIn(), Translator: tr}).
		ListWith(context.Background(), dev, 0)
	s.Require().NoError(err)

	s.Require().Len(listing.Slots, 1)
	s.Require().Equal("Chunky Monkey", listing.Slots[0].Name)
	s.Require().True(listing.Slots[0].Empty(),
		"the double said the slot holds nothing, and it really holds six")
}

func TestTypesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TypesPublicTestSuite))
}
