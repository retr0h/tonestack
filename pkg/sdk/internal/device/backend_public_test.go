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

package device_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// BackendPublicTestSuite covers what a USB backend decides.
type BackendPublicTestSuite struct {
	suite.Suite
}

// thing stands in for whatever a backend lists: a device or an interface.
type thing struct {
	vendor, product uint16
}

func ids(t thing) (vendor, product uint16) { return t.vendor, t.product }

// TestMatching covers keeping the devices asked for.
func (s *BackendPublicTestSuite) TestMatching() {
	all := []thing{{0x0e41, 0x4246}, {0x05ac, 0x1234}, {0x0e41, 0x4253}}

	var released []thing

	kept := device.Matching(all, ids,
		func(vendor, _ uint16) bool { return vendor == 0x0e41 },
		func(t thing) { released = append(released, t) })

	s.Require().Equal([]thing{{0x0e41, 0x4246}, {0x0e41, 0x4253}}, kept)
	// Every reference not kept is given back, or the operating system goes on
	// holding it for this process.
	s.Require().Equal([]thing{{0x05ac, 0x1234}}, released)
}

// TestPickFirst covers choosing one interface out of several.
func (s *BackendPublicTestSuite) TestPickFirst() {
	tests := []struct {
		name     string
		all      []thing
		want     thing
		ok       bool
		released int
	}{
		{name: "one", all: []thing{{1, 1}}, want: thing{1, 1}, ok: true},
		{
			name: "several, the rest given back",
			all:  []thing{{1, 1}, {2, 2}, {3, 3}},
			want: thing{1, 1}, ok: true, released: 2,
		},
		{name: "none"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			released := 0

			got, ok := device.PickFirst(tt.all, func(thing) { released++ })

			s.Require().Equal(tt.ok, ok)
			s.Require().Equal(tt.want, got)
			s.Require().Equal(tt.released, released)
		})
	}
}

// errIdle stands in for a read that timed out with nothing.
var errIdle = errors.New("timed out")

func idle(err error) bool { return errors.Is(err, errIdle) }

// TestReadUntil covers keeping a read posted until the device speaks.
func (s *BackendPublicTestSuite) TestReadUntil() {
	errBroken := errors.New("pipe stalled")

	tests := []struct {
		name string
		// what each read returns, in order.
		reads  []error
		counts []int
		cancel bool
		wantN  int
		want   error
		// how many reads happened.
		calls int
	}{
		{
			// The usual case: nothing, nothing, then an answer.
			name:   "idle reads until the device answers",
			reads:  []error{errIdle, errIdle, nil},
			counts: []int{0, 0, 12},
			wantN:  12, calls: 3,
		},
		{
			name:   "a real error ends the wait",
			reads:  []error{errIdle, errBroken},
			counts: []int{0, 0},
			want:   errBroken, calls: 2,
		},
		{
			// Bytes arrived with the timeout, so they are handed back rather
			// than dropped.
			name:   "a short read that also timed out",
			reads:  []error{errIdle},
			counts: []int{4},
			wantN:  4, want: errIdle, calls: 1,
		},
		{
			// Ctrl-C has to reach a session holding the interface.
			name:   "a cancelled context",
			cancel: true,
			want:   context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, stop := context.WithCancel(context.Background())
			defer stop()

			if tt.cancel {
				stop()
			}

			calls := 0

			read := func(_ []byte, slice time.Duration) (int, error) {
				s.Require().Equal(10*time.Millisecond, slice)

				n, err := tt.counts[calls], tt.reads[calls]
				calls++

				return n, err
			}

			n, err := device.ReadUntil(ctx, make([]byte, 64), 10*time.Millisecond, read, idle)

			s.Require().Equal(tt.wantN, n)
			s.Require().Equal(tt.calls, calls)

			if tt.want == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.want)
		})
	}
}

// TestRefused covers what a failed claim says.
func (s *BackendPublicTestSuite) TestRefused() {
	cause := errors.New("exclusive access")

	busy := device.Refused(cause, true)
	s.Require().ErrorIs(busy, cause)
	s.Require().Contains(busy.Error(), "quit HX Edit")

	other := device.Refused(cause, false)
	s.Require().ErrorIs(other, cause)
	s.Require().NotContains(other.Error(), "HX Edit")
}

// TestLocated covers placing a device by IOKit's location ID.
func (s *BackendPublicTestSuite) TestLocated() {
	s.Require().Equal(device.Descriptor{
		Vendor: 0x0e41, Product: 0x4246, Bus: 0x01, Address: 0x100000,
	}, device.Located(0x0e41, 0x4246, 0x01100000))
}

// TestListed covers describing what is attached.
func (s *BackendPublicTestSuite) TestListed() {
	errBus := errors.New("registry unavailable")

	describe := func(t thing) device.Descriptor {
		return device.Descriptor{Vendor: t.vendor, Product: t.product}
	}

	released := 0

	got, err := device.Listed([]thing{{1, 2}, {3, 4}}, nil, describe,
		func(thing) { released++ })
	s.Require().NoError(err)
	s.Require().Equal([]device.Descriptor{{Vendor: 1, Product: 2}, {Vendor: 3, Product: 4}}, got)
	// Listing keeps nothing.
	s.Require().Equal(2, released)

	_, err = device.Listed[thing](nil, errBus, describe, func(thing) {})
	s.Require().ErrorIs(err, errBus)
}

// TestFound covers keeping the devices asked for as handles.
func (s *BackendPublicTestSuite) TestFound() {
	errBus := errors.New("registry unavailable")
	helix := func(vendor, _ uint16) bool { return vendor == 0x0e41 }

	n, err := device.Found([]thing{{0x0e41, 1}, {0x05ac, 2}}, nil, ids, helix, func(thing) {})
	s.Require().NoError(err)
	s.Require().Equal(1, n)

	_, err = device.Found[thing](nil, errBus, ids, helix, func(thing) {})
	s.Require().ErrorIs(err, errBus)
}

// TestClaimOne covers claiming the editor interface.
func (s *BackendPublicTestSuite) TestClaimOne() {
	errLookup := errors.New("lookup failed")
	errBusy := errors.New("exclusive access")
	errOther := errors.New("not permitted")

	tests := []struct {
		name     string
		ifaces   []thing
		err      error
		open     error
		want     error
		text     string
		released int
	}{
		{name: "claimed", ifaces: []thing{{1, 1}}},
		{name: "the extras given back", ifaces: []thing{{1, 1}, {2, 2}}, released: 1},
		{name: "the lookup failed", err: errLookup, want: errLookup, text: "finding"},
		{name: "no such interface", text: "no interface 0"},
		{
			// HX Edit holds it. The one found is given back, and the message
			// says what to quit.
			name: "held by HX Edit", ifaces: []thing{{1, 1}}, open: errBusy,
			want: errBusy, text: "quit HX Edit", released: 1,
		},
		{
			name: "refused for another reason", ifaces: []thing{{1, 1}}, open: errOther,
			want: errOther, text: "claiming", released: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			released := 0

			got, err := device.ClaimOne(tt.ifaces, tt.err, 0,
				func(thing) { released++ },
				func(thing) error { return tt.open },
				func(err error) bool { return errors.Is(err, errBusy) })

			s.Require().Equal(tt.released, released)

			if tt.text == "" {
				s.Require().NoError(err)
				s.Require().Equal(tt.ifaces[0], got)

				return
			}

			s.Require().ErrorContains(err, tt.text)

			if tt.want != nil {
				s.Require().ErrorIs(err, tt.want)
			}
		})
	}
}

// TestPiped covers turning a pipe lookup into an endpoint.
func (s *BackendPublicTestSuite) TestPiped() {
	errPipe := errors.New("no such endpoint")
	wrap := func(ref uint8) int { return int(ref) }

	got, err := device.Piped(3, nil, wrap)
	s.Require().NoError(err)
	s.Require().Equal(3, got)

	_, err = device.Piped(0, errPipe, wrap)
	s.Require().ErrorIs(err, errPipe)
}

func TestBackendPublicTestSuite(t *testing.T) {
	suite.Run(t, new(BackendPublicTestSuite))
}
