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
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// SessionPublicTestSuite covers a held claim of the pedal: one handshake, many
// operations, one at a time.
type SessionPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *SessionPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
}

// attached is a session that can read, write and change what is playing.
type attached struct {
	*mocks.MockEditor
	*mocks.MockWriter
	*mocks.MockSelector
}

// device is a session that says it is an HX Stomp, and nothing else yet.
func (s *SessionPublicTestSuite) device() *attached {
	dev := &attached{
		MockEditor:   mocks.NewMockEditor(s.ctrl),
		MockWriter:   mocks.NewMockWriter(s.ctrl),
		MockSelector: mocks.NewMockSelector(s.ctrl),
	}
	dev.MockEditor.EXPECT().Model().Return(device.Model{Name: "HX Stomp"}).AnyTimes()

	return dev
}

// over is a Client whose bus opens dev, however often it is asked.
func (s *SessionPublicTestSuite) over(
	dev device.Editor,
) *sdk.Client {
	bus := mocks.NewMockOpener(s.ctrl)
	bus.EXPECT().Open(gomock.Any()).Return(dev, nil).AnyTimes()

	return sdk.New(sdk.WithDevices(bus), sdk.WithBackupDir(s.T().TempDir()))
}

// open is a Session over dev, closed when the test ends.
func (s *SessionPublicTestSuite) open(
	dev *attached,
) *sdk.Session {
	dev.MockEditor.EXPECT().Close().Return(nil).MaxTimes(1)

	session, err := s.over(dev).Open(context.Background())
	s.Require().NoError(err)

	s.T().Cleanup(func() { s.NoError(session.Close()) })

	return session
}

// closed is a Session that has already been closed.
func (s *SessionPublicTestSuite) closed() *sdk.Session {
	dev := s.device()
	dev.MockEditor.EXPECT().Close().Return(nil)

	session, err := s.over(dev).Open(context.Background())
	s.Require().NoError(err)
	s.Require().NoError(session.Close())

	return session
}

// body is a real HX Stomp preset, as the device answers a read.
func (s *SessionPublicTestSuite) body() []byte {
	body, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return body
}

// listing is what the device calls its first two slots.
func listing() []wire.Preset {
	return []wire.Preset{{Slot: 0, Name: "Chunky Monkey"}, {Slot: 1, Name: "Longview"}}
}

// shortly is a context that gives up soon, released when the test ends.
func (s *SessionPublicTestSuite) shortly() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	s.T().Cleanup(cancel)

	return ctx
}

// TestOpen covers claiming the pedal.
func (s *SessionPublicTestSuite) TestOpen() {
	refused := errors.New("no device found")

	tests := []struct {
		name string
		// bus is what the Client reaches the device through.
		bus func() *mocks.MockOpener
		// held opens a Session first and keeps it open while the call runs.
		held bool
		// call is what is tried: an Open, unless the row says otherwise.
		call   func(c *sdk.Client, ctx context.Context) error
		ctx    func() context.Context
		err    error
		says   string
		opened bool
	}{
		{
			name: "a pedal on the bus",
			bus: func() *mocks.MockOpener {
				dev := s.device()
				dev.MockEditor.EXPECT().Close().Return(nil)

				bus := mocks.NewMockOpener(s.ctrl)
				bus.EXPECT().Open(gomock.Any()).Return(dev, nil)

				return bus
			},
			opened: true,
		},
		{
			// The claim is given back, so the next Open reaches the bus
			// rather than waiting on a Session that never was.
			name: "a bus with nothing to open",
			bus: func() *mocks.MockOpener {
				bus := mocks.NewMockOpener(s.ctrl)
				bus.EXPECT().Open(gomock.Any()).Return(nil, refused).Times(2)

				return bus
			},
			call: func(c *sdk.Client, ctx context.Context) error {
				if _, err := c.Open(ctx); !errors.Is(err, refused) {
					return fmt.Errorf("the first open: %w", err)
				}

				_, err := c.Open(ctx)

				return err
			},
			err: refused,
		},
		{
			// No expectations on the bus: reaching it fails the row.
			name: "a caller who stopped waiting",
			bus:  func() *mocks.MockOpener { return mocks.NewMockOpener(s.ctrl) },
			ctx:  cancelled,
			err:  context.Canceled,
		},
		{
			// Two claims of one editor interface are what leaves a pedal
			// needing a power cycle, and the Client is the only thing that
			// knows both exist.
			name: "while another Session from the Client is open",
			bus:  func() *mocks.MockOpener { return nil },
			held: true,
			ctx:  s.shortly,
			err:  context.DeadlineExceeded,
			says: "waiting for the open session to close",
		},
		{
			// A one-shot opens a Session of its own, so it waits the same
			// way.
			name: "a one-shot while a Session from the Client is open",
			bus:  func() *mocks.MockOpener { return nil },
			held: true,
			call: func(c *sdk.Client, ctx context.Context) error {
				_, err := c.Presets(ctx, 0)

				return err
			},
			ctx: s.shortly,
			err: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var (
				client *sdk.Client
				held   *sdk.Session
			)

			if tt.held {
				// Twice: the Session held here, and the one opened once it
				// lets go.
				dev := s.device()
				dev.MockEditor.EXPECT().Close().Return(nil).Times(2)
				client = s.over(dev)

				var err error

				held, err = client.Open(context.Background())
				s.Require().NoError(err)
			} else {
				client = sdk.New(sdk.WithDevices(tt.bus()))
			}

			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			if tt.opened {
				session, err := client.Open(ctx)
				s.Require().NoError(err)
				s.Require().Equal("HX Stomp", session.Model())
				s.Require().NoError(session.Close())

				return
			}

			call := tt.call
			if call == nil {
				call = func(c *sdk.Client, ctx context.Context) error {
					_, err := c.Open(ctx)

					return err
				}
			}

			err := call(client, ctx)
			s.Require().ErrorIs(err, tt.err)
			s.Require().ErrorContains(err, tt.says)

			if !tt.held {
				return
			}

			// Once the first lets go, the next claim goes through.
			s.Require().NoError(held.Close())

			again, err := client.Open(context.Background())
			s.Require().NoError(err)
			s.Require().NoError(again.Close())
		})
	}
}

// TestModel covers which device a Session is talking to.
func (s *SessionPublicTestSuite) TestModel() {
	s.Run("an HX Stomp", func() {
		s.Require().Equal("HX Stomp", s.open(s.device()).Model())
	})
}

// TestPresets covers listing a setlist, and one operation at a time.
func (s *SessionPublicTestSuite) TestPresets() {
	unnamed := []wire.Preset{{Slot: 0, Name: "New Preset"}, {Slot: 1, Name: "New Preset"}}

	s.Run("a setlist", func() {
		dev := s.device()
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 2).Return(unnamed, nil)

		got, err := s.open(dev).Presets(context.Background(), 2)
		s.Require().NoError(err)
		s.Require().Equal("HX Stomp", got.Name)
		s.Require().Len(got.Slots, 2)
	})

	s.Run("after Close", func() {
		_, err := s.closed().Presets(context.Background(), 0)
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})

	s.Run("a caller who stopped waiting", func() {
		// No expectation: the device is never asked.
		_, err := s.open(s.device()).Presets(cancelled(), 0)
		s.Require().ErrorIs(err, context.Canceled)
	})

	s.Run("two callers at once", func() {
		// A barrier rather than a sleep: the first call holds the Session
		// until the test lets it go, and the second must not get in.
		entered, release := make(chan struct{}), make(chan struct{})

		dev := s.device()
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).DoAndReturn(
			func(context.Context, int) ([]wire.Preset, error) {
				close(entered)
				<-release

				return unnamed, nil
			})
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 1).Return(unnamed, nil)

		session := s.open(dev)

		first := make(chan error, 1)

		go func() {
			_, err := session.Presets(context.Background(), 0)
			first <- err
		}()

		<-entered

		_, err := session.Presets(s.shortly(), 1)
		s.Require().ErrorIs(err, context.DeadlineExceeded)
		s.Require().ErrorContains(err, "waiting for the operation in flight")

		close(release)
		s.Require().NoError(<-first)

		_, err = session.Presets(context.Background(), 1)
		s.Require().NoError(err, "the second call gets in once the first lets go")
	})
}

// TestPreset covers reading one slot.
func (s *SessionPublicTestSuite) TestPreset() {
	tests := []struct {
		name   string
		answer []byte
		fails  error
		// empty is a slot that holds nothing, which is an answer.
		empty bool
		says  string
	}{
		{name: "a slot holding a preset", answer: s.body()},
		{name: "a slot holding nothing", empty: true},
		{name: "a device that will not answer", fails: errors.New("no answer"), says: "no answer"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dev := s.device()
			dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil)
			dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 1).Return(tt.answer, tt.fails)

			got, err := s.open(dev).Preset(context.Background(), slot.Address{Slot: 1})

			if tt.says != "" {
				s.Require().ErrorContains(err, tt.says)

				return
			}

			s.Require().NoError(err)

			if tt.empty {
				s.Require().Equal("01B", got.Name)
				s.Require().True(got.Empty())

				return
			}

			s.Require().Equal("Longview", got.Name)
			s.Require().False(got.Empty())
		})
	}

	s.Run("after Close", func() {
		_, err := s.closed().Preset(context.Background(), slot.Address{})
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})
}

// TestExport covers writing one slot out.
func (s *SessionPublicTestSuite) TestExport() {
	tests := []struct {
		name string
		as   sdk.Format
		file string
	}{
		{name: "as a rig", as: sdk.FormatRig, file: "one.yaml"},
		{name: "as the device's own file", as: sdk.FormatPreset, file: "one.hlx"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dev := s.device()
			dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil)
			dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(s.body(), nil)

			out := filepath.Join(s.T().TempDir(), tt.file)

			got, err := s.open(dev).Export(context.Background(), slot.Address{}, out, tt.as)
			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().FileExists(out)
		})
	}

	s.Run("after Close", func() {
		_, err := s.closed().Export(context.Background(), slot.Address{}, "one.yaml", sdk.FormatRig)
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})
}

// TestImport covers putting a preset file into a slot.
func (s *SessionPublicTestSuite) TestImport() {
	s.Run("a preset into a slot", func() {
		dev := s.device()
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil)
		dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, 1).Return(s.body(), nil)
		dev.MockWriter.EXPECT().
			WriteNamedPreset(gomock.Any(), 0, 1, gomock.Any(), gomock.Any()).Return(nil)

		// A preset whose blocks fit the blank an import builds into, under the
		// built-in catalog.
		file := filepath.Join("internal", "compile", "testdata", "preset0.hlx")

		got, err := s.open(dev).Import(context.Background(), file, slot.Address{Slot: 1})
		s.Require().NoError(err)
		s.Require().Equal(sdk.Imported, got.Action)
		s.Require().Len(got.Kept, 1, "what the slot held is kept first")
	})

	s.Run("after Close", func() {
		_, err := s.closed().Import(context.Background(), fixture("preset.hlx"), slot.Address{})
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})
}

// TestCopy covers putting what one slot holds into another.
func (s *SessionPublicTestSuite) TestCopy() {
	s.Run("one slot onto another", func() {
		dev := s.device()
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil)
		dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).
			Return(s.body(), nil).Times(2)
		dev.MockWriter.EXPECT().
			WriteNamedPreset(gomock.Any(), 0, 1, "Chunky Monkey", gomock.Any()).Return(nil)

		got, err := s.open(dev).Copy(
			context.Background(), slot.Address{Slot: 0}, slot.Address{Slot: 1})
		s.Require().NoError(err)
		s.Require().Equal(sdk.Copied, got.Action)
	})

	s.Run("after Close", func() {
		_, err := s.closed().Copy(context.Background(), slot.Address{}, slot.Address{Slot: 1})
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})
}

// TestSwap covers exchanging what two slots hold.
func (s *SessionPublicTestSuite) TestSwap() {
	s.Run("two slots", func() {
		dev := s.device()
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil)
		dev.MockEditor.EXPECT().ReadPreset(gomock.Any(), 0, gomock.Any()).
			Return(s.body(), nil).Times(2)
		dev.MockWriter.EXPECT().
			WriteNamedPreset(gomock.Any(), 0, gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).Times(2)

		got, err := s.open(dev).Swap(
			context.Background(), slot.Address{Slot: 0}, slot.Address{Slot: 1})
		s.Require().NoError(err)
		s.Require().Equal(sdk.Swapped, got.Action)
	})

	s.Run("after Close", func() {
		_, err := s.closed().Swap(context.Background(), slot.Address{}, slot.Address{Slot: 1})
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})
}

// TestSelect covers loading a preset.
func (s *SessionPublicTestSuite) TestSelect() {
	s.Run("a slot", func() {
		dev := s.device()
		dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).Return(listing(), nil)
		dev.MockSelector.EXPECT().SelectPreset(gomock.Any(), 0, 1).Return(nil)

		got, err := s.open(dev).Select(context.Background(), slot.Address{Slot: 1})
		s.Require().NoError(err)
		s.Require().Equal(sdk.Selected, got.Action)
		s.Require().Equal("Longview", got.To.Name)
	})

	s.Run("after Close", func() {
		_, err := s.closed().Select(context.Background(), slot.Address{})
		s.Require().ErrorIs(err, sdk.ErrClosed)
	})
}

// TestClose covers letting the pedal go.
func (s *SessionPublicTestSuite) TestClose() {
	ended := fmt.Errorf("%w: reading from the device: gone", sdk.ErrBus)

	tests := []struct {
		name string
		// released is what the device session says as it is closed.
		released error
		// during runs against the Session before Close, and inflight is an
		// operation still running when Close is called.
		during   func(session *sdk.Session, dev *attached)
		inflight bool
	}{
		{name: "an open Session"},
		{
			// The error that ended the read loop is what Close reports,
			// every time it is asked.
			name:     "one the bus ended",
			released: ended,
		},
		{
			// A panic on the caller's goroutine is not recovered, but the
			// operation's deferred unlock runs as it unwinds, so the caller's
			// deferred Close still lets the pedal go.
			name: "one whose operation panicked",
			during: func(session *sdk.Session, dev *attached) {
				dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).DoAndReturn(
					func(context.Context, int) ([]wire.Preset, error) {
						panic("a bug inside an operation")
					})

				s.Require().Panics(func() {
					_, _ = session.Presets(context.Background(), 0)
				})
			},
		},
		{
			// An operation in flight finishes first. Pulling the interface
			// from under a write is how a pedal gets wedged.
			name:     "one with an operation in flight",
			inflight: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var closes, closedUnder atomic.Int32

			dev := s.device()
			dev.MockEditor.EXPECT().Close().DoAndReturn(func() error {
				closes.Add(1)

				return tt.released
			})

			client := s.over(dev)

			session, err := client.Open(context.Background())
			s.Require().NoError(err)

			if tt.during != nil {
				tt.during(session, dev)
			}

			if tt.inflight {
				entered, release := make(chan struct{}), make(chan struct{})

				dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).DoAndReturn(
					func(context.Context, int) ([]wire.Preset, error) {
						close(entered)
						<-release

						// Read as the operation ends: a Close that did not
						// wait would have let the device go by now.
						closedUnder.Store(closes.Load())

						return nil, nil
					})

				go func() { _, _ = session.Presets(context.Background(), 0) }()

				<-entered

				closing := make(chan error, 1)

				go func() { closing <- session.Close() }()

				close(release)
				s.Require().NoError(<-closing)
				s.Require().Zero(closedUnder.Load(),
					"the device was not let go under the operation")
			} else {
				s.Require().Equal(tt.released, session.Close())
			}

			// Idempotent, and the device is let go once.
			s.Require().Equal(tt.released, session.Close())

			// The claim is given back, so the Client can open again.
			dev.MockEditor.EXPECT().Close().Return(nil)

			again, err := client.Open(s.shortly())
			s.Require().NoError(err)
			s.Require().NoError(again.Close())
		})
	}
}

func TestSessionPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(SessionPublicTestSuite))
}
