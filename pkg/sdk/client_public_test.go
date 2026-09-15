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
	"github.com/retr0h/tonestack/pkg/sdk/slot"
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
		ChainMeasurements(context.Background(), "")

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
				Copy(context.Background(), slot.Address{}, slot.Address{Slot: 1})

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

			_, err := sdk.New(opts...).Preset(context.Background(), slot.Address{})
			s.Require().NoError(err)

			if !tt.asked {
				s.Require().Zero(kept.Len())

				return
			}

			s.Require().Equal(body, kept.Bytes(), "kept verbatim")
		})
	}
}

// TestWithTrace covers where the frames go.
//
// A double sends no frames, so this asks which bus New built: the USB one,
// handed the trace. device's own tests show a session writes its frames there.
func (s *ClientPublicTestSuite) TestWithTrace() {
	var trace bytes.Buffer

	tests := []struct {
		name  string
		opts  []sdk.Option
		trace io.Writer
	}{
		{name: "asked for", opts: []sdk.Option{sdk.WithTrace(&trace)}, trace: &trace},
		{name: "not asked for"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(device.NewUSB(tt.trace), sdk.New(tt.opts...).Opener())
		})
	}
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
		name    string
		ctx     context.Context
		catalog string
		id      string
		is      error
		err     bool
	}{
		{name: "a block the catalog carries", id: "HD2_DistMinotaur"},
		{name: "one it does not", id: "HD2_NoSuchBlock", is: sdk.ErrNoSuchBlock},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			id:   "HD2_DistMinotaur",
			is:   context.Canceled,
		},
		{
			name:    "a catalog that is not there",
			catalog: "no.json",
			id:      "HD2_DistMinotaur",
			err:     true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New(sdk.WithCatalog(tt.catalog)).Block(ctx, tt.id)

			if tt.err {
				s.Require().Error(err)

				return
			}

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, string(got.ID))
		})
	}
}

// TestModelMeasurements covers how players set one model.
func (s *ClientPublicTestSuite) TestModelMeasurements() {
	tests := []struct {
		name  string
		ctx   context.Context
		model string
		err   bool
	}{
		{name: "one model's distributions", model: "HD2_DistMinotaur"},
		{name: "a model nobody measured", model: "HD2_NoSuchModel", err: true},
		{
			// Refused rather than answered with the grammar of a chain,
			// which is a different question with a method of its own.
			name: "no model at all",
			err:  true,
		},
		{
			name:  "a caller who stopped waiting",
			ctx:   cancelled(),
			model: "HD2_DistMinotaur",
			err:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().ModelMeasurements(ctx, tt.model)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(got.Stats)
			s.Require().True(got.AboutOne())
			s.Require().Equal(tt.model, string(got.Model))
		})
	}
}

// TestChainMeasurements covers what chains tend to hold.
func (s *ClientPublicTestSuite) TestChainMeasurements() {
	tests := []struct {
		name       string
		ctx        context.Context
		instrument string
		err        bool
	}{
		{name: "every chain"},
		{name: "one instrument's chains", instrument: "bass"},
		{name: "a caller who stopped waiting", ctx: cancelled(), err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := sdk.New().ChainMeasurements(ctx, tt.instrument)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(got.Stats)
			s.Require().False(got.AboutOne())
			s.Require().Equal(tt.instrument, got.Instrument)
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
	scaffold := func(ctx context.Context, in sdk.NewRecipe, opts ...sdk.Option) (scaffolded, error) {
		got, err := sdk.New(opts...).Scaffold(ctx, in)
		if err != nil {
			return scaffolded{}, err
		}

		body, err := os.ReadFile(got.Path)
		s.Require().NoError(err)

		return scaffolded{got: got, body: string(body)}, nil
	}

	// Each row writes into a directory of its own, because a scaffold is
	// never written over a rig already there.
	into := func() sdk.Option { return sdk.WithRecipes(s.T().TempDir()) }

	base := sdk.NewRecipe{
		ID: "test-player", Name: "Test Player",
		Instrument: "bass", Amp: "Ampeg SVT",
	}

	written, err := scaffold(context.Background(), base, into())
	s.Require().NoError(err)

	// with is the base recipe with one field changed.
	with := func(change func(*sdk.NewRecipe)) sdk.NewRecipe {
		in := base
		change(&in)

		return in
	}

	tests := []struct {
		name string
		ctx  context.Context
		in   sdk.NewRecipe
		// nowhere means the Client was given no directory of rigs.
		nowhere bool
		// check is what the written rig must say that the base rig does
		// not, for a row that changes one field.
		check func(got scaffolded)
		err   bool
	}{
		{
			name: "a rig naming gear this device models",
			in:   base,
			check: func(got scaffolded) {
				s.Require().Equal(base.ID, got.got.ID)
				s.Require().Equal(written.body, got.body)
			},
		},
		{
			name: "Name decides who the rig is about",
			in:   with(func(in *sdk.NewRecipe) { in.Name = "Other Player" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "  name: Other Player")
				s.Require().Contains(written.body, "  name: Test Player")
			},
		},
		{
			name: "Band decides the group the rig names",
			in:   with(func(in *sdk.NewRecipe) { in.Band = "The Test Band" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "  band: The Test Band")
				s.Require().NotContains(written.body, "band:")
			},
		},
		{
			name: "Instrument decides which instrument the rig is for",
			in:   with(func(in *sdk.NewRecipe) { in.Instrument = "guitar" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "instrument: guitar")
				s.Require().Contains(written.body, "instrument: bass")
			},
		},
		{
			name: "Amp decides the amplifier in the chain",
			in:   with(func(in *sdk.NewRecipe) { in.Amp = "Acoustic 360" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "role: amp\n    gear: Acoustic 360")
				s.Require().Contains(written.body, "role: amp\n    gear: Ampeg SVT")
			},
		},
		{
			name: "Cab decides the cabinet in the chain",
			in:   with(func(in *sdk.NewRecipe) { in.Cab = "Ampeg 8x10" }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "role: cab\n    gear: Ampeg 8x10")
				s.Require().NotContains(written.body, "role: cab")
			},
		},
		{
			name: "Pedals decide what goes ahead of the amp",
			in:   with(func(in *sdk.NewRecipe) { in.Pedals = []string{"Klon"} }),
			check: func(got scaffolded) {
				s.Require().Contains(got.body, "role: drive\n    gear: Klon")
				s.Require().NotContains(written.body, "role: drive")
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
				opts = append(opts, into())
			}

			got, err := scaffold(ctx, tt.in, opts...)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			tt.check(got)
		})
	}
}

// scaffolded is what a scaffold answered and the rig it wrote.
type scaffolded struct {
	got  sdk.Scaffolded
	body string
}

// extended is what an extend answered and the rig it wrote.
type extended struct {
	got  sdk.Scaffolded
	body string
}

// TestExtend covers starting a rig as a copy of another.
//
// ExtendRecipe is an input struct, so each of its fields has a row of its own
// that changes that field alone and shows the rig written changing with it. A
// field no path reads would leave its row identical to the baseline.
func (s *ClientPublicTestSuite) TestExtend() {
	extend := func(ctx context.Context, in sdk.ExtendRecipe, opts ...sdk.Option) (extended, error) {
		got, err := sdk.New(opts...).Extend(ctx, in)
		if err != nil {
			return extended{}, err
		}

		body, err := os.ReadFile(got.Path)
		s.Require().NoError(err)

		return extended{got: got, body: string(body)}, nil
	}

	// Each row writes into a directory of its own, because a copy is never
	// written over a rig already there.
	into := func() sdk.Option { return sdk.WithRecipes(s.T().TempDir()) }

	base, err := extend(
		context.Background(),
		sdk.ExtendRecipe{From: "mike-dirnt", ID: "the-copy"},
		into(),
	)
	s.Require().NoError(err)

	tests := []struct {
		name string
		ctx  context.Context
		in   sdk.ExtendRecipe
		// nowhere means the Client was given no directory of rigs.
		nowhere bool
		check   func(got extended)
		is      error
		err     bool
	}{
		{
			name: "From decides which rig is copied",
			in:   sdk.ExtendRecipe{From: "flea", ID: "the-copy"},
			check: func(got extended) {
				s.Require().Contains(got.body, "extends: flea")
				s.Require().Contains(base.body, "extends: mike-dirnt")
				s.Require().NotEqual(base.body, got.body)
			},
		},
		{
			name: "ID decides what the copy is called and where it is written",
			in:   sdk.ExtendRecipe{From: "mike-dirnt", ID: "another-copy"},
			check: func(got extended) {
				s.Require().Equal("another-copy", got.got.ID)
				s.Require().Equal("another-copy.yaml", filepath.Base(got.got.Path))
				s.Require().Contains(got.body, "id: another-copy")
				s.Require().NotContains(base.body, "id: another-copy")
			},
		},
		{
			name: "Name decides who the copy is about",
			in:   sdk.ExtendRecipe{From: "mike-dirnt", ID: "the-copy", Name: "Somebody Else"},
			check: func(got extended) {
				s.Require().Contains(got.body, "  name: Somebody Else")
				s.Require().Contains(base.body, "  name: Mike Dirnt")
			},
		},
		{
			name: "Kind decides what the copy is attributed to",
			in:   sdk.ExtendRecipe{From: "mike-dirnt", ID: "the-copy", Kind: "song"},
			check: func(got extended) {
				s.Require().Contains(got.body, "  kind: song")
				s.Require().Contains(base.body, "  kind: artist")
			},
		},
		{
			name: "a rig to copy that nobody wrote",
			in:   sdk.ExtendRecipe{From: "nobody-at-all", ID: "the-copy"},
			is:   sdk.ErrNoSuchRecipe,
		},
		{
			// Without one this would scaffold a rig from no gear at all.
			name: "no rig to copy",
			in:   sdk.ExtendRecipe{ID: "the-copy"},
			is:   sdk.ErrNoSuchRecipe,
		},
		{
			// Refused rather than written wherever the program happened to
			// run.
			name:    "a Client given nowhere to write",
			in:      sdk.ExtendRecipe{From: "mike-dirnt", ID: "the-copy"},
			nowhere: true,
			err:     true,
		},
		{
			name: "a caller who stopped waiting",
			ctx:  cancelled(),
			in:   sdk.ExtendRecipe{From: "mike-dirnt", ID: "the-copy"},
			is:   context.Canceled,
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
				opts = append(opts, into())
			}

			got, err := extend(ctx, tt.in, opts...)

			switch {
			case tt.is != nil:
				s.Require().ErrorIs(err, tt.is)

				return
			case tt.err:
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			tt.check(got)
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

			got, err := sdk.New().Build(ctx, tt.id, out)

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

func TestClientPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(ClientPublicTestSuite))
}
