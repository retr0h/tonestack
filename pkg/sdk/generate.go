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

package sdk

import (
	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogen"
	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusgen"
)

// DefaultResourcesDir is where HX Edit installs its model definitions.
//
// Offered so a caller does not have to know the layout of somebody else's
// application bundle to ask for a catalog.
const DefaultResourcesDir = catalogen.DefaultResourcesDir

// Catalog says what to build a device catalog from.
//
// The inputs are a licensed HX Edit installation and the gear map extracted
// from its Pilot's Guide, so this is a thing a maintainer runs and not a thing
// a preset needs. It is here because what it writes is the catalog this
// library embeds, and the code that makes a file belongs with the file.
type Catalog struct {
	// ResourcesDir is an HX Edit installation to read model definitions from.
	ResourcesDir string
	// GearMapPath says which real-world gear each model emulates.
	GearMapPath string
	// SourceName records which release this came from. A catalog is only
	// true of the one it was extracted from, so it says which.
	SourceName string
	// DeviceName and DeviceID name the hardware it describes.
	DeviceName string
	DeviceID   int
	// SchemaVersion is the catalog format to write. Zero writes the current
	// one.
	SchemaVersion int
	// OutputPath is where the catalog goes.
	OutputPath string
}

// GenerateCatalog builds a device catalog and writes it.
func (c *Client) GenerateCatalog(in Catalog) (Catalogued, error) {
	return catalogen.Run(catalogen.Options{
		ResourcesDir:  in.ResourcesDir,
		GearMapPath:   in.GearMapPath,
		SourceName:    in.SourceName,
		DeviceName:    in.DeviceName,
		DeviceID:      in.DeviceID,
		SchemaVersion: in.SchemaVersion,
		OutputPath:    in.OutputPath,
	})
}

// Measure says what to measure a corpus of presets against.
type Measure struct {
	// CorpusDir holds the presets to measure.
	CorpusDir string
	// CatalogPath is a catalog to read instead of the built-in one.
	CatalogPath string
	// MinSamples is how many values a parameter needs before its
	// distribution is worth keeping. Zero takes the default.
	MinSamples int
	// OutputPath is where the statistics go.
	OutputPath string
}

// MeasureCorpus measures what real presets say and writes the statistics.
func (c *Client) MeasureCorpus(in Measure) (Counted, error) {
	return corpusgen.Run(corpusgen.Options{
		CorpusDir:   in.CorpusDir,
		CatalogPath: in.CatalogPath,
		MinSamples:  in.MinSamples,
		OutputPath:  in.OutputPath,
	})
}
