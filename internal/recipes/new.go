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

package recipes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/catalog"
)

// Sentinels callers match with errors.Is.
var (
	// ErrBadID reports an identifier that will not do.
	ErrBadID = errors.New("bad identifier")
	// ErrExists reports a recipe that is already there.
	ErrExists = errors.New("recipe already exists")
	// ErrNoSuchGear reports gear this device does not model.
	ErrNoSuchGear = errors.New("no such gear")
)

// idPattern is the shape an identifier takes: lower case words joined by
// hyphens, matching the filename it is written to.
var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// BadIDError says what was wrong with an identifier.
type BadIDError struct {
	ID string
}

func (e *BadIDError) Error() string {
	return fmt.Sprintf(
		"bad identifier: %q — use lower case words joined by hyphens, like mike-dirnt",
		e.ID)
}

func (*BadIDError) Unwrap() error { return ErrBadID }

// ExistsError names the file already there.
type ExistsError struct {
	Path string
}

func (e *ExistsError) Error() string {
	return fmt.Sprintf("recipe already exists: %s", e.Path)
}

func (*ExistsError) Unwrap() error { return ErrExists }

// NoSuchGearError names gear the device does not model, and what it does.
type NoSuchGearError struct {
	Want  string
	Near  []string
	Field string
}

func (e *NoSuchGearError) Error() string {
	msg := fmt.Sprintf("no such gear: nothing this device models emulates %q", e.Want)
	if len(e.Near) > 0 {
		msg += "\n  did you mean: " + strings.Join(e.Near, ", ")
	}

	return msg
}

func (*NoSuchGearError) Unwrap() error { return ErrNoSuchGear }

// NewOptions describes the recipe to scaffold.
type NewOptions struct {
	// Dir is where recipes live. Empty writes beside the built-in ones,
	// which is not usually what anybody wants.
	Dir string
	// ID is the identifier, and the filename stem.
	ID string
	// Name is the player or style, as a person would write it.
	Name string
	// Band is the group, where there is one.
	Band string
	// Instrument is guitar or bass.
	Instrument string
	// Amp is the real-world amplifier. Required: it is the one thing nothing
	// downstream recovers from getting wrong.
	Amp string
	// Cab is the real-world cabinet. Empty takes the amp's own pairing.
	Cab string
	// Pedals are real-world pedals, in signal order.
	Pedals []string
	// CatalogPath is a catalog to check against instead of the built-in one.
	CatalogPath string
	// From is a rig to copy, by identifier. The copy is a whole rig and
	// records where it came from in `extends`; nothing merges the two.
	From string
	// Kind is what the new rig is attributed to: artist, band, song, genre
	// or sound. Only read when copying, since a scaffold from nothing is an
	// artist.
	Kind string
}

// New writes a recipe, after checking the gear it names exists.
//
// Checking first is the point. A recipe naming gear no device models is only
// discovered when somebody tries to build from it, and by then the name has
// usually been copied somewhere else too.
func New(w io.Writer, opts NewOptions) error {
	if !idPattern.MatchString(opts.ID) {
		return &BadIDError{ID: opts.ID}
	}

	body, err := scaffoldFor(opts)
	if err != nil {
		return err
	}

	path := filepath.Join(opts.Dir, "artists", opts.ID+".yaml")
	if _, err := os.Stat(path); err == nil {
		return &ExistsError{Path: path}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("making room for %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return report(w, opts, path)
}

// checkGear refuses a recipe naming gear the device has no model for.
func checkGear(cat *catalog.Catalog, opts NewOptions) error {
	type gear struct {
		field string
		want  string
	}

	named := make([]gear, 0, 2+len(opts.Pedals))
	named = append(named, gear{"amp", opts.Amp}, gear{"cab", opts.Cab})

	for _, p := range opts.Pedals {
		named = append(named, gear{"pedal", p})
	}

	for _, n := range named {
		if n.want == "" {
			continue
		}

		if !models(cat, n.want) {
			return &NoSuchGearError{
				Want: n.want, Field: n.field, Near: near(cat, n.want),
			}
		}
	}

	return nil
}

// models reports whether anything in the catalog is this gear.
func models(cat *catalog.Catalog, want string) bool {
	for _, b := range cat.Blocks {
		if b.Matches(want) {
			return true
		}
	}

	return false
}

// near suggests gear whose name shares a word with what was asked for.
//
// Not fuzzy matching: a shared word is a strong enough signal to be worth
// showing, and a weak one is worse than nothing when somebody is deciding
// whether they got the name wrong.
func near(cat *catalog.Catalog, want string) []string {
	words := strings.Fields(strings.ToLower(want))
	seen := map[string]bool{}

	for _, b := range cat.Blocks {
		if b.BasedOn == "" {
			continue
		}

		lower := strings.ToLower(b.BasedOn)

		for _, word := range words {
			if len(word) > 2 && strings.Contains(lower, word) {
				seen[b.BasedOn] = true
			}
		}
	}

	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}

	sort.Strings(out)

	if len(out) > 5 {
		out = out[:5]
	}

	return out
}

// report says what was written and what to do with it.
func report(w io.Writer, opts NewOptions, path string) error {
	rows := [][]string{
		{cli.Mute(w, "id"), cli.Accent(w, opts.ID)},
		{cli.Mute(w, "instrument"), opts.Instrument},
		{cli.Mute(w, "amp"), opts.Amp},
	}

	if opts.Cab != "" {
		rows = append(rows, []string{cli.Mute(w, "cab"), opts.Cab})
	}

	if len(opts.Pedals) > 0 {
		rows = append(rows,
			[]string{cli.Mute(w, "pedals"), strings.Join(opts.Pedals, ", ")})
	}

	if err := (cli.Section{
		Title: opts.Name, Detail: path, Rows: rows,
		Summary: fmt.Sprintf(
			"every gear name resolves — next: tonestack presets make --id %s --out %s.hlx",
			opts.ID, opts.ID),
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// scaffoldFor decides what goes in the new file.
//
// Copying an existing rig checks nothing, because the rig it copies already
// resolved when it was written and the copy has not changed any gear yet.
// Scaffolding from flags checks every name against the catalog, which is the
// only moment a typo is cheap to fix.
func scaffoldFor(opts NewOptions) (string, error) {
	if opts.From != "" {
		parent, from, err := findFile(opts.Dir, opts.From)
		if err != nil {
			return "", err
		}

		return scaffold(parent, from, opts), nil
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return "", err
	}

	if err := checkGear(cat, opts); err != nil {
		return "", err
	}

	return render(opts), nil
}
