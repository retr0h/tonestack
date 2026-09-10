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
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// oneGoodWrite fails only after the first write, so a caller that says one
// thing before another reports the second one's failure rather than the first.
type oneGoodWrite struct{ n int }

func (w *oneGoodWrite) Write(p []byte) (int, error) {
	w.n++
	if w.n > 1 {
		return 0, io.ErrClosedPipe
	}

	return len(p), nil
}

type ChangePublicTestSuite struct {
	suite.Suite
}

// TestChange covers saying what a write did.
func (s *ChangePublicTestSuite) TestChange() {
	from := sdk.At{Slot: 0, Name: "First"}

	tests := []struct {
		name string
		in   sdk.Change
		to   io.Writer
		want []string
		err  bool
	}{
		{
			// A file says where it was written, because there is somewhere
			// to go and look.
			name: "one slot copied into another, in a file",
			in: sdk.Change{
				Action:   sdk.Copied,
				From:     &from,
				To:       sdk.At{Slot: 1, Name: "Second"},
				Replaced: "Second",
				Path:     "/tmp/out.hls",
			},
			want: []string{"01A", "First", "01B", "copied", "wrote /tmp/out.hls"},
		},
		{
			// A device says what the destination stopped being, because
			// there is no file and the backup is the only way back.
			name: "one slot copied into another, on a device",
			in: sdk.Change{
				Action:   sdk.Swapped,
				From:     &from,
				To:       sdk.At{Slot: 1, Name: "Second"},
				Replaced: "Second",
				Kept:     []string{"/tmp/01B.hlx"},
			},
			want: []string{"kept", "/tmp/01B.hlx", "swapped, replacing Second"},
		},
		{
			name: "a preset written into a slot on a device",
			in: sdk.Change{
				Action: sdk.Imported,
				To:     sdk.At{Slot: 6, Name: "Mike Dirnt"},
				Kept:   []string{"/tmp/07A.hlx"},
			},
			want: []string{"kept", "Mike Dirnt", "03A", "written"},
		},
		{
			name: "a preset written into a slot in a file",
			in: sdk.Change{
				Action:   sdk.Imported,
				To:       sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Replaced: "Second",
				Path:     "/tmp/out.hls",
			},
			want: []string{"01B", "Mike Dirnt", "replaced", "Second", "wrote"},
		},
		{
			// Loading writes nothing, so there is nothing to say about what
			// the slot held: it still holds it.
			name: "a preset loaded, which writes nothing",
			in: sdk.Change{
				Action: sdk.Selected,
				To:     sdk.At{Slot: 4, Name: "Montana"},
			},
			want: []string{"02B", "Montana", "loaded"},
		},
		{
			name: "a preset made for another device",
			in: sdk.Change{
				Action:   sdk.Imported,
				To:       sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Replaced: "Second",
				Path:     "/tmp/out.hls",
				Mismatch: true,
			},
			want: []string{"different device", "may not load"},
		},
		{
			// A slot that held nothing has no backup, and a line saying so
			// would leave somebody wondering what it was for.
			name: "a backup of a slot holding nothing",
			in: sdk.Change{
				Action: sdk.Imported,
				To:     sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Kept:   []string{""},
			},
			want: []string{"Mike Dirnt"},
		},
		{
			name: "nowhere to say a backup went",
			in: sdk.Change{
				Action: sdk.Imported,
				To:     sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Kept:   []string{"/tmp/01B.hlx"},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to say what moved",
			in: sdk.Change{
				Action: sdk.Copied,
				From:   &from,
				To:     sdk.At{Slot: 1, Name: "Second"},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to say what landed on a device",
			in: sdk.Change{
				Action: sdk.Imported,
				To:     sdk.At{Slot: 1, Name: "Mike Dirnt"},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to say what landed in a file",
			in: sdk.Change{
				Action: sdk.Imported,
				To:     sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Path:   "/tmp/out.hls",
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to say what was loaded",
			in: sdk.Change{
				Action: sdk.Selected,
				To:     sdk.At{Slot: 4, Name: "Montana"},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			// The warning itself is what fails, before anything else has
			// been said.
			name: "nowhere to say it was made for another device",
			in: sdk.Change{
				Action:   sdk.Imported,
				To:       sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Path:     "/tmp/out.hls",
				Mismatch: true,
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			// The warning goes out and the line after it does not, which is
			// the second write failing rather than the first.
			name: "nowhere to say what landed, after the warning",
			in: sdk.Change{
				Action:   sdk.Imported,
				To:       sdk.At{Slot: 1, Name: "Mike Dirnt"},
				Path:     "/tmp/out.hls",
				Mismatch: true,
			},
			to:  &oneGoodWrite{},
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Change(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestBuilt covers saying what was compiled from a rig.
func (s *ChangePublicTestSuite) TestBuilt() {
	tests := []struct {
		name string
		in   sdk.Built
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "a chain of several blocks",
			in:   sdk.Built{Name: "Mike Dirnt", Blocks: 3, Path: "/tmp/one.hlx"},
			want: []string{"Mike Dirnt", "3 blocks in the chain", "wrote /tmp/one.hlx"},
		},
		{
			// A chain of one reads as "1 blocks" without this.
			name: "a chain of one block",
			in:   sdk.Built{Name: "Mike Dirnt", Blocks: 1, Path: "/tmp/one.hlx"},
			want: []string{"1 block in the chain"},
		},
		{
			name: "nowhere to say it",
			in:   sdk.Built{Name: "Mike Dirnt", Blocks: 3, Path: "/tmp/one.hlx"},
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

			err := cli.Built(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

func TestChangePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ChangePublicTestSuite))
}
