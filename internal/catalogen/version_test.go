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

package catalogen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type VersionTestSuite struct {
	suite.Suite
}

func (s *VersionTestSuite) TestAppVersionReadsTheBundle() {
	got := AppVersion(filepath.Join("testdata", "bundle", "Resources"))

	s.Require().Equal("9.99", got)
}

func (s *VersionTestSuite) TestAppVersionIsEmptyWhenItCannotTell() {
	tests := []struct {
		name string
		dir  string
	}{
		{
			"model definitions that are not inside an application bundle",
			filepath.Join("testdata", "families"),
		},
		{"a directory that is not there", filepath.Join("testdata", "nope", "x")},
		{
			"a bundle whose property list names no version",
			filepath.Join("testdata", "noversion", "Resources"),
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Empty(AppVersion(tc.dir),
				"a missing version is recorded as unknown, not treated as a failure")
		})
	}
}

func (s *VersionTestSuite) TestPlistString() {
	tests := []struct {
		name    string
		plist   string
		key     string
		want    string
		wantErr string
	}{
		{
			"a key that is there",
			`<plist><dict><key>A</key><string>one</string></dict></plist>`,
			"A", "one", "",
		},
		{
			"a later key, past one that does not match",
			`<plist><dict><key>A</key><string>one</string>` +
				`<key>B</key><string>two</string></dict></plist>`,
			"B", "two", "",
		},
		{
			"a key whose value is not a string",
			`<plist><dict><key>A</key><integer>1</integer>` +
				`<key>B</key><string>two</string></dict></plist>`,
			"B", "two", "",
		},
		{
			"whitespace between the elements",
			"<plist>\n <dict>\n  <key>A</key>\n  <string>one</string>\n" +
				" </dict>\n</plist>",
			"A", "one", "",
		},
		{
			"a key nobody wrote",
			`<plist><dict><key>A</key><string>one</string></dict></plist>`,
			"Missing", "", "Missing is not in this plist",
		},
		{"a document that will not parse", `<plist><dict`, "A", "", "XML syntax"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, err := PlistString(strings.NewReader(tc.plist), tc.key)

			if tc.wantErr != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.wantErr)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tc.want, got)
		})
	}
}

func (s *VersionTestSuite) TestSourceNamesTheRelease() {
	s.Require().Equal("Test Editor 9.99", sourceName(Options{
		ResourcesDir: filepath.Join("testdata", "bundle", "Resources"),
		SourceName:   "Test Editor",
	}))

	s.Require().Empty(sourceName(Options{
		ResourcesDir: filepath.Join("testdata", "families"),
		SourceName:   "Test Editor",
	}))
}

func TestVersionTestSuite(t *testing.T) {
	suite.Run(t, new(VersionTestSuite))
}
