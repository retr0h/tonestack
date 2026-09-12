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
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// versionKey is the Info.plist key holding the version a person would
// recognise — "3.82" rather than a build number.
const versionKey = "CFBundleShortVersionString"

// appVersion reads the version of the application the resources belong to.
//
// The .models files carry no version of their own, and the app bundle's
// Info.plist is the only place the release is written down. A catalog that
// cannot name its source cannot be checked against a device later, so this is
// read rather than assumed.
//
// Missing is not an error. Model definitions can be pointed at from somewhere
// that is not an application bundle, and refusing to build a catalog over a
// version string would be pedantry. The catalog records what was found, and
// an empty result reads as "source unknown" wherever it is shown.
func appVersion(resourcesDir string) string {
	// Resources sits beside Info.plist inside Contents.
	f, err := os.Open(
		filepath.Join(resourcesDir, "..", "Info.plist"),
	) //nolint:gosec // a path the caller chose
	if err != nil {
		return ""
	}

	defer func() { _ = f.Close() }()

	v, err := plistString(f, versionKey)
	if err != nil {
		return ""
	}

	return v
}

// plistString pulls one string value out of an XML property list.
//
// A plist dictionary is a flat run of <key> and value elements rather than
// nested pairs, so the value wanted is the first <string> after the matching
// <key>. Decoding the whole document into a struct is not possible for the
// same reason.
func plistString(r io.Reader, key string) (string, error) {
	dec := xml.NewDecoder(r)

	var (
		inKey   bool
		inValue bool
		armed   bool
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return "", fmt.Errorf("%s is not in this plist", key)
		}

		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			inKey = t.Name.Local == "key"
			inValue = armed && t.Name.Local == "string"
		case xml.EndElement:
			// Whitespace between elements arrives as character data too, so
			// the flags are cleared here rather than left to be reset by
			// whatever comes next.
			inKey, inValue = false, false
		case xml.CharData:
			switch {
			case inKey:
				armed = string(t) == key
			case inValue:
				return string(t), nil
			}
		}
	}
}
