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

package deviceslots_test

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
	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	slotmocks "github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// TypesPublicTestSuite covers standing something else in for a collaborator.
//
// Every other suite here leaves the collaborators on Flows nil and gets the
// real thing, which is what the Client does in earnest. These drive the other
// half: a flow given a double uses it, and what it reports is the double's.
type TypesPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *TypesPublicTestSuite) SetupTest() { s.ctrl = gomock.NewController(s.T()) }

func (s *TypesPublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// builtIn is a double handing over the catalog in this binary.
func (s *TypesPublicTestSuite) builtIn() *slotmocks.MockCatalogs {
	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	cat := slotmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Catalog(gomock.Any()).Return(built, nil)

	return cat
}

// answer returns one slot as the hardware sent it.
func (s *TypesPublicTestSuite) answer() []byte {
	raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

// TestCompiler covers reading a slot into a rig through a double.
func (s *TypesPublicTestSuite) TestCompiler() {
	want := errors.New("cannot lift that")

	dev := mocks.NewMockEditor(s.ctrl)
	dev.EXPECT().Presets(gomock.Any(), 0).
		Return([]wire.Preset{{Slot: 0, Name: "Chunky Monkey"}}, nil)
	dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer(), nil)

	comp := slotmocks.NewMockCompiler(s.ctrl)
	comp.EXPECT().Lift(gomock.Any(), gomock.Any()).Return(rig.Spec{}, want)

	_, err := (&deviceslots.Flows{Catalogs: s.builtIn(), Compiler: comp}).
		Show(context.Background(), dev, slotpkg.Address{})

	s.Require().ErrorIs(err, want)
}

// TestTranslator covers a listing reading each slot's chain through a double.
func (s *TypesPublicTestSuite) TestTranslator() {
	dev := mocks.NewMockEditor(s.ctrl)
	dev.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.EXPECT().Presets(gomock.Any(), 0).
		Return([]wire.Preset{{Slot: 0, Name: "Chunky Monkey"}}, nil)
	dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer(), nil)

	// The slot really holds six blocks. The double says it holds none, so a
	// listing that shows it empty can only have asked the double.
	tr := slotmocks.NewMockTranslator(s.ctrl)
	tr.EXPECT().Chain("", gomock.Any(), gomock.Any()).Return(chain.Chain{}, nil)

	listing, err := (&deviceslots.Flows{Catalogs: s.builtIn(), Translator: tr}).
		List(context.Background(), dev, 0)
	s.Require().NoError(err)

	s.Require().Len(listing.Slots, 1)
	s.Require().Equal("Chunky Monkey", listing.Slots[0].Name)
	s.Require().True(listing.Slots[0].Empty(),
		"the double said the slot holds nothing, and it really holds six")
}

// TestBackups covers a write asking a double to keep what it replaces, and
// not writing when the double fails.
func (s *TypesPublicTestSuite) TestBackups() {
	tests := []struct {
		name string
		// what the keeper answers.
		kept []string
		err  error
		// the write goes out.
		writes bool
	}{
		{
			name:   "a keeper that kept the slot",
			kept:   []string{"somewhere/03B.hlx"},
			writes: true,
		},
		{
			// A write whose backup failed does not happen. No write is
			// expected on the device, so one fails the row.
			name: "a keeper that failed",
			err:  errors.New("no room"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dev := &writable{
				MockEditor: mocks.NewMockEditor(s.ctrl),
				MockWriter: mocks.NewMockWriter(s.ctrl),
			}
			dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
				Return([]wire.Preset{{Slot: 0, Name: "Chunky Monkey"}, {Slot: 7, Name: "Minor Threat"}}, nil)
			dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.answer(), nil)
			dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 7).Return(s.answer(), nil)

			keeper := slotmocks.NewMockBackups(s.ctrl)
			keeper.EXPECT().
				Keep(gomock.Any(), backup.Held{
					At: slotpkg.Address{Slot: 7}, Name: "Minor Threat", Body: s.answer(),
				}).
				Return(tt.kept, tt.err)

			if tt.writes {
				dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 7, "Chunky Monkey", s.answer()).
					Return(nil)
			}

			change, err := (&deviceslots.Flows{Backups: keeper}).
				Copy(context.Background(), dev, slotpkg.Address{}, slotpkg.Address{Slot: 7})

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.kept, change.Kept)
		})
	}
}

func TestTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TypesPublicTestSuite))
}
