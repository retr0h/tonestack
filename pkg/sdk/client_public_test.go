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

package sdk_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/device"
)

// bus stands in for the one thing this library needs hardware for.
type bus struct {
	descs []device.Descriptor
	err   error
}

func (b *bus) List(context.Context) ([]device.Descriptor, error) {
	return b.descs, b.err
}

func (*bus) Close() error { return nil }

type ClientPublicTestSuite struct {
	suite.Suite
}

// stand puts a bus in front of the Client and gives back what undoes it.
func (s *ClientPublicTestSuite) stand(b *bus) func() {
	restore := *sdk.NewLister
	*sdk.NewLister = func() sdk.Closer { return b }

	return func() { *sdk.NewLister = restore }
}

// TestNew covers building a Client.
func (s *ClientPublicTestSuite) TestNew() {
	tests := []struct {
		name string
		opts []sdk.Option
	}{
		{
			// The common case is somebody who wants the built-in catalog,
			// the built-in statistics and whatever is plugged in.
			name: "with nothing said about it",
		},
		{
			name: "with an option that says nothing yet",
			opts: []sdk.Option{func(*sdk.Options) {}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().NotNil(sdk.New(tt.opts...))
		})
	}
}

// TestDevices covers reporting what is attached.
func (s *ClientPublicTestSuite) TestDevices() {
	stomp := device.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 20, Address: 3}

	tests := []struct {
		name  string
		bus   *bus
		want  int
		first string
		err   bool
	}{
		{
			name:  "a device this project knows",
			bus:   &bus{descs: []device.Descriptor{stomp}},
			want:  1,
			first: "HX Stomp",
		},
		{
			// A bus holds keyboards and webcams. Those are not an answer to
			// what a preset can be written to.
			name: "somebody else's hardware",
			bus:  &bus{descs: []device.Descriptor{{Vendor: 0x05ac, Product: 0x1234}}},
		},
		{name: "nothing attached", bus: &bus{}},
		{
			name: "a bus that will not answer",
			bus:  &bus{err: errors.New("bus unavailable")},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer s.stand(tt.bus)()

			found, err := sdk.New().Devices(context.Background())

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(found.Devices, tt.want)

			if tt.first != "" {
				s.Require().Equal(tt.first, found.Devices[0].Model)
			}
		})
	}
}

// TestBlocks covers reporting what a device can do.
func (s *ClientPublicTestSuite) TestBlocks() {
	tests := []struct {
		name    string
		path    string
		filter  sdk.Filter
		matched bool
		err     bool
	}{
		{
			// No path means the catalog in the binary, which is what anyone
			// who has not generated their own wants.
			name:    "the catalog in the binary",
			filter:  sdk.Filter{Search: "klon"},
			matched: true,
		},
		{
			name:   "a filter nothing matches",
			filter: sdk.Filter{Category: "no such category"},
		},
		{name: "a catalog that is not there", path: "no.json", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.New().Blocks(tt.path, tt.filter)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotZero(got.Total)

			if tt.matched {
				s.Require().NotEmpty(got.Matched)

				return
			}

			s.Require().Empty(got.Matched)
		})
	}
}

// TestBlock covers reporting one block and what it accepts.
func (s *ClientPublicTestSuite) TestBlock() {
	tests := []struct {
		name string
		id   string
		err  bool
	}{
		{name: "a block the catalog carries", id: "HD2_DistMinotaur"},
		{name: "one it does not", id: "HD2_NoSuchBlock", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.New().Block("", tt.id)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, string(got.ID))
		})
	}
}

// TestMeasurements covers both questions the corpus answers.
func (s *ClientPublicTestSuite) TestMeasurements() {
	tests := []struct {
		name     string
		in       sdk.Corpus
		aboutOne bool
		err      bool
	}{
		{
			// Naming no model asks what chains of a kind are shaped like.
			name: "the grammar of a chain",
		},
		{
			name:     "one model's distributions",
			in:       sdk.Corpus{Model: "HD2_DistMinotaur"},
			aboutOne: true,
		},
		{
			name: "a model nobody measured",
			in:   sdk.Corpus{Model: "HD2_NoSuchModel"},
			err:  true,
		},
		{
			name: "statistics that are not there",
			in:   sdk.Corpus{StatsPath: "no.json.gz"},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.New().Measurements(tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(got.Stats)
			s.Require().Equal(tt.aboutOne, got.AboutOne())
		})
	}
}

// TestRecipes covers reading the rigs that ship with this library.
func (s *ClientPublicTestSuite) TestRecipes() {
	tests := []struct {
		name string
		dir  string
		// a shelf with nothing on it, which is an answer rather than a
		// failure: somebody who just made the directory is owed one.
		empty bool
	}{
		{
			// No directory means the rigs that ship, which is the case for
			// anyone who has not written their own.
			name: "the rigs that ship",
		},
		{name: "a shelf that is not there", dir: "no-such-directory", empty: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.New().Recipes(tt.dir)

			s.Require().NoError(err)
			s.Require().Equal(tt.dir, got.Dir)

			if tt.empty {
				s.Require().Empty(got.Rigs)

				return
			}

			s.Require().NotEmpty(got.Rigs)
		})
	}
}

// TestRecipe covers reading one of them.
func (s *ClientPublicTestSuite) TestRecipe() {
	tests := []struct {
		name string
		id   string
		err  bool
	}{
		{name: "a rig that ships", id: "mike-dirnt"},
		{name: "one nobody wrote", id: "nobody-at-all", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.New().Recipe("", tt.id)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, got.Rig.ID)
		})
	}
}

// TestScaffold covers writing a rig, gear checked first.
func (s *ClientPublicTestSuite) TestScaffold() {
	tests := []struct {
		name string
		in   sdk.NewRecipe
		err  bool
	}{
		{
			name: "a rig naming gear this device models",
			in: sdk.NewRecipe{
				ID: "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "Ampeg SVT",
			},
		},
		{
			// Checked first, because a rig naming gear no device models is
			// otherwise only found out when somebody builds from it.
			name: "one naming gear nothing emulates",
			in: sdk.NewRecipe{
				ID: "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "No Such Amplifier",
			},
			err: true,
		},
		{
			name: "an identifier that will not do",
			in:   sdk.NewRecipe{ID: "Not An ID", Amp: "Ampeg SVT"},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			in := tt.in
			in.Dir = s.T().TempDir()

			got, err := sdk.New().Scaffold(in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.in.ID, got.ID)
			s.Require().FileExists(got.Path)
		})
	}
}

// TestBuild covers compiling a rig into a preset.
func (s *ClientPublicTestSuite) TestBuild() {
	tests := []struct {
		name string
		id   string
		err  bool
	}{
		{name: "a rig that ships", id: "mike-dirnt"},
		{name: "one nobody wrote", id: "nobody-at-all", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "out.hlx")

			got, err := sdk.New().Build(sdk.Make{RecipeID: tt.id, OutputPath: out})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().NotEmpty(got.Chain.Blocks)
		})
	}
}

func TestClientPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ClientPublicTestSuite))
}
