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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
	sdk "github.com/retr0h/tonestack/pkg/sdk"
)

// BackingPublicTestSuite covers the table that holds a rig's records to its
// era.
type BackingPublicTestSuite struct {
	suite.Suite
}

// render draws a set of rigs and hands back what was written.
func (s *BackingPublicTestSuite) render(
	all []sdk.Backing,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Backing(&buf, all))

	return buf.String()
}

// TestRecordsThatMatchTheEra covers the case nobody has to act on.
func (s *BackingPublicTestSuite) TestRecordsThatMatchTheEra() {
	got := s.render([]sdk.Backing{{
		ID: "geddy-lee", Era: "Fly by Night through Hemispheres",
		From: 1975, To: 1978,
		Records: []sdk.Record{
			{Track: "anthem", Year: 1975},
			{Track: "la-villa-strangiato", Year: 1978},
		},
	}})

	s.Require().Contains(got, "geddy-lee")
	s.Require().Contains(got, "1975–1978")
	s.Require().Contains(got, "records match the era")
	s.Require().Contains(got, "every record in era")
}

// TestEveryRecordFromAnotherEra is the finding this table exists for.
func (s *BackingPublicTestSuite) TestEveryRecordFromAnotherEra() {
	got := s.render([]sdk.Backing{{
		ID: "flea", Era: "2012 touring", From: 2012, To: 2012,
		Records: []sdk.Record{
			{Track: "aeroplane", Year: 1995, Outside: true},
			{Track: "suck-my-kiss", Year: 1991, Outside: true},
		},
	}})

	s.Require().Contains(got, "every record is from another era")
	s.Require().Contains(got, "2012")
	s.Require().Contains(got, "1 measuring records from another era")
}

// TestSomeRecordsFromAnotherEra covers the partial case, which is the one
// somebody can fix by swapping a record.
func (s *BackingPublicTestSuite) TestSomeRecordsFromAnotherEra() {
	got := s.render([]sdk.Backing{{
		ID: "tim-commerford", From: 1992, To: 1992,
		Records: []sdk.Record{
			{Track: "bombtrack", Year: 1992},
			{Track: "sleep-now-in-the-fire", Year: 1999, Outside: true},
		},
	}})

	s.Require().Contains(got, "1 of 2 from another era")
	s.Require().Contains(got, "1992")
}

// TestARigWithNoEra covers what cannot be checked, which is worth seeing.
func (s *BackingPublicTestSuite) TestARigWithNoEra() {
	got := s.render([]sdk.Backing{{
		ID:      "somebody",
		Records: []sdk.Record{{Track: "one", Year: 1999}},
	}})

	s.Require().Contains(got, "says none")
	s.Require().Contains(got, "no era to hold them to")
}

// TestARigNobodyHasMeasured covers the ordinary case: gear evidence long
// before anybody owns the records.
func (s *BackingPublicTestSuite) TestARigNobodyHasMeasured() {
	got := s.render([]sdk.Backing{{ID: "somebody", From: 1994, To: 1994}})

	s.Require().Contains(got, "none measured")
	s.Require().Contains(got, "nothing measured for it")
}

// TestOneYearReadsAsOneYear covers a rig that applied for a single year.
func (s *BackingPublicTestSuite) TestOneYearReadsAsOneYear() {
	got := s.render([]sdk.Backing{{
		ID: "pino-palladino", From: 2000, To: 2000,
		Records: []sdk.Record{{Track: "chicken-grease", Year: 2000}},
	}})

	s.Require().Contains(got, "2000")
	s.Require().NotContains(got, "2000–2000")
}

// TestNoRigsAtAll covers the empty table.
func (s *BackingPublicTestSuite) TestNoRigsAtAll() {
	got := s.render(nil)

	s.Require().Contains(got, "no rigs to read")
}

func TestBackingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BackingPublicTestSuite))
}
