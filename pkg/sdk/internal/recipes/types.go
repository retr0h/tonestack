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
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Source says where rigs are read from.
//
// The zero value is the rigs that ship in the binary.
type Source struct {
	// Dir is a directory read instead of the rigs that ship. Empty is the
	// rigs that ship.
	Dir string
	// User is somebody's own directory, layered over Dir. A rig in it takes
	// the place of one in Dir when the two share an identifier or alias, in
	// any case. A directory that is not there holds nothing, and one that
	// cannot be read is an error. Empty layers nothing.
	User string
}

// stored is one rig as read, and the text it was read from. The text is kept
// because a copy keeps the comments and decoding drops them.
type stored struct {
	spec rig.Spec
	raw  []byte
}

// brokenFile is a file in somebody's own directory that is not a rig.
type brokenFile struct {
	// names are what the file may have been asked for by: its filename stem,
	// and whatever id and aliases it states where those can be read.
	names []string
	err   error
}

// set is every rig a Source holds, read once.
type set struct {
	user   []stored
	base   []stored
	broken []brokenFile
}

// Catalogs hands over the catalog a new rig's gear is checked against. The
// sdk Client satisfies it, and keeps the catalog it opened.
type Catalogs interface {
	// Catalog returns the catalog, opening it on first use.
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}
