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
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

type ClientPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *ClientPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// cancelled is a context whose caller has already stopped waiting.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

// bus stands in for the one thing this library needs hardware for.
func (s *ClientPublicTestSuite) bus(
	descs []device.Descriptor,
	err error,
) *mocks.MockOpener {
	b := mocks.NewMockOpener(s.ctrl)
	b.EXPECT().List(gomock.Any()).Return(descs, err).AnyTimes()

	return b
}

// writable is a session that can both read and write.
type writable struct {
	*mocks.MockEditor
	*mocks.MockWriter
}

// pedal is a bus whose one device holds a real HX Stomp preset in every slot,
// and takes whatever is written to it.
func (s *ClientPublicTestSuite) pedal() *mocks.MockOpener {
	body, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	dev := &writable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockWriter: mocks.NewMockWriter(s.ctrl),
	}
	dev.MockEditor.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()
	dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return([]wire.Preset{
		{Slot: 0, Name: "Chunky Monkey"},
		{Slot: 1, Name: "Longview"},
	}, nil).AnyTimes()
	dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).
		Return(body, nil).AnyTimes()
	dev.MockEditor.EXPECT().Close().AnyTimes()
	dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	b := mocks.NewMockOpener(s.ctrl)
	b.EXPECT().Open(gomock.Any()).Return(dev, nil).AnyTimes()

	return b
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
			// Nothing is opened, so even settings naming nothing real
			// build a Client. What they name is found out on use.
			name: "with every setting",
			opts: []sdk.Option{
				sdk.WithCatalog("no.json"),
				sdk.WithStats("no.json.gz"),
				sdk.WithRecipes("no-such-directory"),
				sdk.WithBackupDir("no-such-directory"),
				sdk.WithCapture(io.Discard),
				sdk.WithTrace(io.Discard),
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().NotNil(sdk.New(tt.opts...))
		})
	}
}

// TestWithCatalog covers naming a catalog other than the built-in one.
func (s *ClientPublicTestSuite) TestWithCatalog() {
	builtIn, err := sdk.New().Blocks(context.Background(), sdk.Filter{})
	s.Require().NoError(err)

	tests := []struct {
		name string
		path string
		err  bool
	}{
		{name: "a catalog of its own", path: fixture("catalog.json")},
		{name: "a catalog that is not there", path: "no.json", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.New(sdk.WithCatalog(tt.path)).
				Blocks(context.Background(), sdk.Filter{})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEqual(builtIn.Total, got.Total,
				"the Client read the catalog it was given")
		})
	}
}

// TestWithStats covers naming statistics other than the built-in ones.
func (s *ClientPublicTestSuite) TestWithStats() {
	_, err := sdk.New(sdk.WithStats("no.json.gz")).
		Measurements(context.Background(), sdk.Corpus{})

	s.Require().Error(err, "the Client read the statistics it was given")
}

// TestWithRecipes covers naming a directory of rigs.
func (s *ClientPublicTestSuite) TestWithRecipes() {
	got, err := sdk.New(sdk.WithRecipes("no-such-directory")).
		Recipes(context.Background())

	s.Require().NoError(err)
	s.Require().Equal("no-such-directory", got.Dir)
	s.Require().Empty(got.Rigs)
}

// TestWithBackupDir covers where a device slot's old contents go.
func (s *ClientPublicTestSuite) TestWithBackupDir() {
	tests := []struct {
		name string
		dir  func() string
		err  bool
	}{
		{
			name: "a directory it can write",
			dir:  func() string { return s.T().TempDir() },
		},
		{
			// A write whose backup failed does not happen.
			name: "somewhere nothing can be kept",
			dir: func() string {
				path := filepath.Join(s.T().TempDir(), "a-file")
				s.Require().NoError(os.WriteFile(path, nil, 0o600))

				return path
			},
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := tt.dir()

			got, err := sdk.New(sdk.WithDevices(s.pedal()), sdk.WithBackupDir(dir)).
				Copy(context.Background(), sdk.Edit{FromSlot: 0, ToSlot: 1})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got.Kept, 1)
			s.Require().Equal(dir, filepath.Dir(got.Kept[0]))
		})
	}
}

// TestWithCapture covers keeping what a device answered.
func (s *ClientPublicTestSuite) TestWithCapture() {
	body, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	tests := []struct {
		name  string
		asked bool
	}{
		{name: "when asked", asked: true},
		{name: "when nobody asked"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var kept bytes.Buffer

			opts := []sdk.Option{sdk.WithDevices(s.pedal())}
			if tt.asked {
				opts = append(opts, sdk.WithCapture(&kept))
			}

			_, err := sdk.New(opts...).Preset(context.Background(), sdk.Read{Slot: 0})
			s.Require().NoError(err)

			if !tt.asked {
				s.Require().Zero(kept.Len())

				return
			}

			s.Require().Equal(body, kept.Bytes(), "kept verbatim")
		})
	}
}

// TestWithTrace covers where the frames go. Only a real bus sends any, so all
// that can be said without one is that asking for them changes no answer.
func (s *ClientPublicTestSuite) TestWithTrace() {
	var trace bytes.Buffer

	got, err := sdk.New(sdk.WithTrace(&trace), sdk.WithDevices(s.bus(nil, nil))).
		Devices(context.Background())

	s.Require().NoError(err)
	s.Require().Empty(got.Devices)
	s.Require().Zero(trace.Len(), "a double sends no frames to trace")
}

// TestCatalog covers the catalog a Client names gear against.
func (s *ClientPublicTestSuite) TestCatalog() {
	tests := []struct {
		name string
		path string
		ctx  context.Context
		err  bool
	}{
		{name: "the catalog in the binary"},
		{name: "a catalog of its own", path: fixture("catalog.json")},
		{name: "a catalog that is not there", path: "no.json", err: true},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			client := sdk.New(sdk.WithCatalog(tt.path))

			got, err := client.Catalog(ctx)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			// Opened once and kept, so a renderer reads what the operation did.
			again, err := client.Catalog(ctx)
			s.Require().NoError(err)
			s.Require().Same(got, again)
		})
	}
}

// TestDevices covers reporting what is attached.
func (s *ClientPublicTestSuite) TestDevices() {
	stomp := device.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 20, Address: 3}

	tests := []struct {
		name  string
		bus   *mocks.MockOpener
		want  int
		first string
		err   bool
	}{
		{
			name:  "a device this project knows",
			bus:   s.bus([]device.Descriptor{stomp}, nil),
			want:  1,
			first: "HX Stomp",
		},
		{
			// A bus holds keyboards and webcams. Those are not an answer to
			// what a preset can be written to.
			name: "somebody else's hardware",
			bus:  s.bus([]device.Descriptor{{Vendor: 0x05ac, Product: 0x1234}}, nil),
		},
		{name: "nothing attached", bus: s.bus(nil, nil)},
		{
			name: "a bus that will not answer",
			bus:  s.bus(nil, errors.New("bus unavailable")),
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			found, err := sdk.New(sdk.WithDevices(tt.bus)).Devices(context.Background())

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
		ctx     context.Context
		filter  sdk.Filter
		matched bool
		err     bool
	}{
		{
			name:    "the catalog in the binary",
			filter:  sdk.Filter{Search: "klon"},
			matched: true,
		},
		{
			name:   "a filter nothing matches",
			filter: sdk.Filter{Category: "no such category"},
		},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Blocks(ctx, tt.filter)

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
		ctx  context.Context
		id   string
		is   error
	}{
		{name: "a block the catalog carries", id: "HD2_DistMinotaur"},
		{name: "one it does not", id: "HD2_NoSuchBlock", is: sdk.ErrNoSuchBlock},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			id:   "HD2_DistMinotaur",
			is:   context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Block(ctx, tt.id)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

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
		ctx      context.Context
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
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Measurements(ctx, tt.in)

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
		ctx  context.Context
		err  bool
	}{
		{
			// No directory means the rigs that ship, which is the case for
			// anyone who has not written their own.
			name: "the rigs that ship",
		},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Recipes(ctx)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotEmpty(got.Rigs)
		})
	}
}

// TestRecipe covers reading one of them.
func (s *ClientPublicTestSuite) TestRecipe() {
	tests := []struct {
		name string
		ctx  context.Context
		id   string
		is   error
	}{
		{name: "a rig that ships", id: "mike-dirnt"},
		{name: "one nobody wrote", id: "nobody-at-all", is: sdk.ErrNoSuchRecipe},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			id:   "mike-dirnt",
			is:   context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().Recipe(ctx, tt.id)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

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
		ctx  context.Context
		in   sdk.NewRecipe
		// nowhere means the Client was given no directory of rigs.
		nowhere bool
		err     bool
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
		{
			// Refused rather than written wherever the program happened to
			// run.
			name: "a Client given nowhere to write",
			in: sdk.NewRecipe{
				ID: "test-player", Name: "Test Player",
				Instrument: "bass", Amp: "Ampeg SVT",
			},
			nowhere: true,
			err:     true,
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			in:   sdk.NewRecipe{ID: "test-player", Amp: "Ampeg SVT"},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			var opts []sdk.Option
			if !tt.nowhere {
				opts = append(opts, sdk.WithRecipes(s.T().TempDir()))
			}

			got, err := sdk.New(opts...).Scaffold(ctx, tt.in)

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
		ctx  context.Context
		id   string
		err  bool
	}{
		{name: "a rig that ships", id: "mike-dirnt"},
		{name: "one nobody wrote", id: "nobody-at-all", err: true},
		{name: "a caller who stopped waiting", ctx: cancelled(), id: "mike-dirnt", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			out := filepath.Join(s.T().TempDir(), "out.hlx")

			got, err := sdk.New().Build(ctx, sdk.Make{RecipeID: tt.id, OutputPath: out})

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
	t.Parallel()

	suite.Run(t, new(ClientPublicTestSuite))
}
