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

package cli_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// brokenWriter fails every write, so a reporting failure is reported rather
// than swallowed.
type brokenWriter struct{}

func (*brokenWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// valid is the smallest rig the contract accepts, so a test about rendering
// one is not also a test about what a rig must carry.
func valid() riggen.RigSpec {
	v := riggen.RigSpecVersion(2)

	return riggen.RigSpec{
		Schema:     "RigSpec",
		Version:    &v,
		ID:         "lead",
		Subject:    riggen.Subject{Kind: "artist", Name: "Lead"},
		Instrument: "bass",
		Chain:      []riggen.ChainEntry{{Gear: "Ampeg SVT", Role: "amp"}},
	}
}

type ReadingPublicTestSuite struct {
	suite.Suite
}

func (s *ReadingPublicTestSuite) TestReading() {
	tests := []struct {
		name string
		read sdk.Reading
		to   io.Writer
		want string
		err  bool
	}{
		{
			name: "a slot holding nothing says so",
			read: sdk.Reading{Name: "Empty"},
			want: "# Empty is empty\n",
		},
		{
			name: "a reply that was not a preset is reported",
			read: sdk.Reading{
				Name: "Lead",
				Answer: &sdk.Answer{
					Model: "HX Stomp",
					Slot:  3,
					Shape: "map with 2 keys",
				},
			},
			want: "map with 2 keys",
		},
		{
			name: "a rig is written as the document it is",
			read: sdk.Reading{Name: "Lead", Doc: &preset.Document{}, Rig: valid()},
			want: "gear: Ampeg SVT",
		},
		{
			name: "a rig that does not validate is reported, not printed empty",
			read: sdk.Reading{Name: "Lead", Doc: &preset.Document{}},
			err:  true,
		},
		{
			name: "nowhere to say a slot holds nothing",
			read: sdk.Reading{Name: "Empty"},
			to:   &brokenWriter{},
			err:  true,
		},
		{
			// The rig validated and rendered; the sink is what failed. That
			// is the one thing left that can go wrong here, so it is the one
			// thing worth reporting.
			name: "nowhere to write the rig",
			read: sdk.Reading{Name: "Lead", Doc: &preset.Document{}, Rig: valid()},
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Reading(to, tt.read)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(buf.String(), tt.want)
		})
	}
}

func (s *ReadingPublicTestSuite) TestAnswer() {
	tests := []struct {
		name  string
		shape string
		to    io.Writer
		want  string
		err   bool
	}{
		{
			name:  "what the device answered with",
			shape: "map with 2 keys",
			want:  "map with 2 keys",
		},
		{
			name:  "nowhere to say it",
			shape: "map with 2 keys",
			to:    &brokenWriter{},
			err:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Answer(to, sdk.Answer{
				Model: "HX Stomp",
				Slot:  3,
				Shape: tt.shape,
			})

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(buf.String(), tt.want)
			s.Require().Contains(buf.String(), "slot 02A")
		})
	}
}

func (s *ReadingPublicTestSuite) TestWritten() {
	tests := []struct {
		name    string
		written sdk.Written
		to      io.Writer
		want    string
		err     bool
	}{
		{
			name:    "the slot, the name and the path",
			written: sdk.Written{Slot: 3, Name: "Lead", Path: "/tmp/lead.yaml"},
			want:    "wrote /tmp/lead.yaml",
		},
		{
			name:    "nowhere to say it",
			written: sdk.Written{Slot: 3, Name: "Lead", Path: "/tmp/lead.yaml"},
			to:      &brokenWriter{},
			err:     true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Written(to, tt.written)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(buf.String(), tt.want)
			s.Require().Contains(buf.String(), "02A")
			s.Require().Contains(buf.String(), "Lead")
		})
	}
}

func TestReadingPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadingPublicTestSuite))
}
