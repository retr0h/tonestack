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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// DecoderPublicTestSuite covers reading a device's answer as the preset a
// backup keeps.
type DecoderPublicTestSuite struct {
	suite.Suite
}

// answer returns one slot as an HX Stomp sent it.
func (s *DecoderPublicTestSuite) answer() []byte {
	raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

// emptied returns a preset with nothing on the grid, as the device sends one.
func (s *DecoderPublicTestSuite) emptied() []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeString("l6-helix\x00"))
	s.Require().NoError(enc.EncodeString("offsets"))
	s.Require().NoError(enc.Encode(map[int8]any{0: map[int8]any{22: []any{}}}))

	return buf.Bytes()
}

// TestDocument covers the three answers a backup can be handed.
func (s *DecoderPublicTestSuite) TestDocument() {
	tests := []struct {
		name string
		body []byte
		// what the device calls the slot.
		called string
		// a document comes back, named for the slot.
		doc     bool
		errText string
	}{
		{
			name:   "a slot holding a preset",
			body:   s.answer(),
			called: "Black Rusty",
			doc:    true,
		},
		{
			// No document, which is how a backup tells a slot with no blocks
			// from one that holds something.
			name: "a slot holding no blocks",
			body: s.emptied(),
		},
		{
			name:    "an answer that is not a preset",
			body:    []byte("nonsense"),
			errText: "reading slot 02A",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := deviceslots.NewDecoder(&deviceslots.Flows{}).
				Document(context.Background(), tt.body, slotpkg.Address{Slot: 3}, tt.called)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
				s.Require().Nil(got)

				return
			}

			s.Require().NoError(err)

			if !tt.doc {
				s.Require().Nil(got)

				return
			}

			s.Require().NotNil(got)
			s.Require().Equal(tt.called, got.Data.Meta.Name)
		})
	}
}

func TestDecoderPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DecoderPublicTestSuite))
}
