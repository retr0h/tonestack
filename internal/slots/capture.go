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
	"io"
	"os"

	"github.com/retr0h/tonestack/internal/cli"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// dumpEnv names a file to write a device's raw answer to.
//
// Reading a preset off the hardware is the one call whose reply nobody has
// seen. Capturing it is what turns a guess about the wire format into a test,
// and it costs one plugged-in session rather than one per attempt.
const dumpEnv = "TONESTACK_USB_DUMP"

// dump writes a device's answer where somebody can read it, when asked.
func dump(got any) error {
	path := os.Getenv(dumpEnv)
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

// describe reports what came back, in whatever detail can be had.
//
// The reply is a document nobody has decoded yet: a preset arrives as three
// concatenated MessagePack values, and the models inside it are numbered by a
// scheme that does not index the catalog. Until that is worked out, saying
// plainly what arrived beats printing a chain that would be wrong.
func describe(w io.Writer, model string, slot int, got any) error {
	shape := fmt.Sprintf("%T", got)

	if m, ok := got.(map[any]any); ok {
		shape = fmt.Sprintf("map with %d keys", len(m))
	}

	if b, ok := got.([]byte); ok {
		shape = fmt.Sprintf("%d bytes", len(b))
	}

	// A preset arrives as an opaque run of bytes that MessagePack's string
	// type happens to carry. Reporting it as a string would suggest text.
	if str, ok := got.(string); ok {
		shape = fmt.Sprintf("%d bytes", len(str))
	}

	return cli.Section{
		Title:   model,
		Detail:  "slot " + slotpkg.Label(slot),
		Headers: []string{"the device answered"},
		Rows:    [][]string{{shape}},
		Summary: fmt.Sprintf("set %s to a path to keep it", dumpEnv),
	}.Render(w)
}
