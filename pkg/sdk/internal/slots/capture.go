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

package slots

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// dump writes a device's answer where somebody can read it, when asked.
func dump(got any) error {
	path := os.Getenv(result.DumpEnv)
	if path == "" {
		return nil
	}

	body, err := bytesOf(got)
	if err != nil {
		return fmt.Errorf("encoding the device's answer: %w", err)
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// bytesOf renders a device's answer for writing to disk.
//
// A preset comes back as an opaque run of bytes, and MessagePack's string type
// is what carries it — the device does not mean text by that, and it is not
// valid UTF-8. Writing it verbatim is the whole point: encoding it as JSON
// replaces every byte above 0x7f with U+FFFD, which destroys exactly the
// offsets and model numbers this capture exists to study.
//
// Anything else is a decoded document, and JSON is the readable way to keep
// one.
func bytesOf(got any) ([]byte, error) {
	switch v := got.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return json.MarshalIndent(got, "", "  ")
	}
}
