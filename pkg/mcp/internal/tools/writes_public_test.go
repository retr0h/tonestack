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

package tools_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

type WritesPublicTestSuite struct {
	suite.Suite
	ctrl   *gomock.Controller
	client *mocks.MockClient
}

func (s *WritesPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
	s.client = mocks.NewMockClient(s.ctrl)
}

// run drives one tool through the table. Every row that succeeds is expected
// to answer an sdk.Change carrying the Replaced value the mock returned,
// which is checked here rather than per row.
func (s *WritesPublicTestSuite) run(
	tool string,
	tests []deviceRow,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			pedal := mocks.NewMockSession(s.ctrl)

			if tt.setup != nil {
				tt.setup(s.client, pedal)
			}

			held(s.client, pedal)

			res := call(s.T(), connect(s.T(), s.client, true), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if !tt.err {
				var got sdk.Change
				structured(s.T(), res, &got)
				s.Equal("Old Preset", got.Replaced)
			}
		})
	}
}

// TestPresetImport covers putting a file into a slot.
func (s *WritesPublicTestSuite) TestPresetImport() {
	s.run("preset_import", []deviceRow{
		{
			name: "a preset into a slot",
			args: tools.Put{Preset: "mike.hlx", Slot: "01A"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Import(gomock.Any(), "mike.hlx", slot.Address{}).
					Return(sdk.Change{Replaced: "Old Preset"}, nil)
			},
			want: "put mike.hlx into 01A",
		},
		{
			name: "a label the pedal does not have",
			args: tools.Put{Preset: "mike.hlx", Slot: "nope"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Put{Preset: "mike.hlx", Slot: "01A"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetsCopy covers copying a slot.
func (s *WritesPublicTestSuite) TestPresetsCopy() {
	s.run("presets_copy", []deviceRow{
		{
			name: "one slot onto another",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Copy(gomock.Any(), slot.Address{}, slot.Address{Slot: 1}).
					Return(sdk.Change{Replaced: "Old Preset"}, nil)
			},
			want: "copied 01A to 01B",
		},
		{
			name: "a source that does not parse",
			args: tools.Move{From: "nope", To: "01B"},
			err:  true,
		},
		{
			name: "a destination that does not parse",
			args: tools.Move{From: "01A", To: "nope"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Move{From: "01A", To: "01B"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

// TestPresetsSwap covers exchanging two slots.
func (s *WritesPublicTestSuite) TestPresetsSwap() {
	s.run("presets_swap", []deviceRow{
		{
			name: "two slots",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(_ *mocks.MockClient, pedal *mocks.MockSession) {
				pedal.EXPECT().Swap(gomock.Any(), slot.Address{}, slot.Address{Slot: 1}).
					Return(sdk.Change{Replaced: "Old Preset"}, nil)
			},
			want: "swapped 01A and 01B",
		},
		{
			name: "a source that does not parse",
			args: tools.Move{From: "nope", To: "01B"},
			err:  true,
		},
		{
			name: "a destination that does not parse",
			args: tools.Move{From: "01A", To: "nope"},
			err:  true,
		},
		{
			name:  "HX Edit holding the pedal",
			args:  tools.Move{From: "01A", To: "01B"},
			setup: hxEdit,
			want:  "quit HX Edit",
			err:   true,
		},
	})
}

func TestWritesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WritesPublicTestSuite))
}
