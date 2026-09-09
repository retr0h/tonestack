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

// TestAppVersion reads which release of HX Edit a catalog came from.
func (s *VersionTestSuite) TestAppVersion() {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "an application bundle",
			dir:  filepath.Join("testdata", "bundle", "Resources"),
			want: "9.99",
		},
		{
			// A missing version is recorded as unknown, not treated as a
			// failure.
			name: "model definitions outside a bundle",
			dir:  filepath.Join("testdata", "families"),
		},
		{
			name: "a directory that is not there",
			dir:  filepath.Join("testdata", "nope", "x"),
		},
		{
			name: "a bundle whose property list names no version",
			dir:  filepath.Join("testdata", "noversion", "Resources"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, AppVersion(tt.dir))
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

// TestSourceName names the release a catalog was generated from.
func (s *VersionTestSuite) TestSourceName() {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "an installation that says which it is",
			dir:  filepath.Join("testdata", "bundle", "Resources"),
			want: "Test Editor 9.99",
		},
		{
			name: "one that does not",
			dir:  filepath.Join("testdata", "families"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, sourceName(Options{
				ResourcesDir: tt.dir, SourceName: "Test Editor",
			}))
		})
	}
}

func TestVersionTestSuite(t *testing.T) {
	suite.Run(t, new(VersionTestSuite))
}
