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

package rig

import (
	"encoding/json"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

// Load reads a rig and checks it against its own contract.
//
// YAML, because a rig is written and corrected by hand and JSON is a poor
// format to argue with. The schema is JSON Schema either way; sigs.k8s.io/yaml
// converts, so the generated types need no second set of tags.
func Load(r io.Reader) (Spec, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Spec{}, fmt.Errorf("reading rig: %w", err)
	}

	// The file as it was written, checked before anything decodes it.
	//
	// Decoding drops whatever the generated types have no field for, so a rig
	// checked after decoding is checked with its own mistakes already
	// removed: `tecnique:` for `technique:` would be dropped on the way in,
	// the rig would pass, and the line somebody wrote would be gone with
	// nothing said.
	body, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return Spec{}, fmt.Errorf("decoding rig: %w", err)
	}

	// What YAMLToJSON produced is JSON, so reading it back cannot fail.
	var document any
	_ = json.Unmarshal(body, &document)

	if err := against(document); err != nil {
		return Spec{}, err
	}

	// Checked rather than assumed. A document can satisfy the contract and
	// still not fit the types: JSON Schema calls 99999999999999999999 an
	// integer and Go's int cannot hold it, so this returned a rig with the
	// field silently zeroed and no error at all.
	var spec Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		return Spec{}, fmt.Errorf("decoding rig: %w", err)
	}

	return spec, nil
}

// Write renders a rig.
//
// Validated first: writing one that does not meet its own contract would put
// a file into the world that nothing else will accept.
func Write(w io.Writer, spec Spec) error {
	if err := Validate(spec); err != nil {
		return err
	}

	// A rig holds only strings, numbers and booleans, so it always encodes.
	raw, _ := yaml.Marshal(spec)

	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("writing rig: %w", err)
	}

	return nil
}
