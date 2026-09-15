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

package fileslots_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	slotmocks "github.com/retr0h/tonestack/pkg/sdk/internal/fileslots/mocks"
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

// TestCatalogs covers a flow reaching its catalog through a double, and the
// one built into this binary when it is given none.
func (s *TypesPublicTestSuite) TestCatalogs() {
	tests := []struct {
		name string
		// what a double answers. Nil gives the flows no double at all.
		err error
	}{
		{name: "a double that cannot open one", err: errors.New("no catalog here")},
		{name: "no double, so the built-in one"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := &fileslots.Flows{}

			if tt.err != nil {
				cat := slotmocks.NewMockCatalogs(s.ctrl)
				cat.EXPECT().Catalog(gomock.Any()).Return(nil, tt.err)

				f.Catalogs = cat
			}

			read, err := f.ShowFile(context.Background(), s.preset())

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(read.Rig.Chain)
		})
	}
}

// TestCompiler covers a flow reading a preset into a rig through a double.
func (s *TypesPublicTestSuite) TestCompiler() {
	tests := []struct {
		name string
		// what the lift answers.
		lifted rig.Spec
		err    error
		call   func(*fileslots.Flows, string) error
		says   string
	}{
		{
			name: "a lift that fails",
			err:  errors.New("cannot lift that"),
			call: func(f *fileslots.Flows, _ string) error {
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
			call: func(f *fileslots.Flows, out string) error {
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

			err := tt.call(&fileslots.Flows{Catalogs: s.builtIn(), Compiler: comp}, out)

			s.Require().ErrorContains(err, tt.says)
			s.Require().NoFileExists(out)
		})
	}
}

func TestTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TypesPublicTestSuite))
}
