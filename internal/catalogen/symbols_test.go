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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// SymbolsTestSuite covers the table a device names its models by.
type SymbolsTestSuite struct {
	suite.Suite
}

// write puts a symbol file in a fresh directory and returns it.
func (s *SymbolsTestSuite) write(body string) string {
	dir := s.T().TempDir()
	s.Require().NoError(
		os.WriteFile(filepath.Join(dir, symbolFile), []byte(body), 0o600))

	return dir
}

func (s *SymbolsTestSuite) TestReadsTheTable() {
	got, err := readSymbols(s.write(`[
		{ "symbol": "HD2_AmpSVBeastNrm",
		  "parameters": [ "Drive", "Bass", "Mid" ] },
		{ "symbol": "HD2_Cab8x10SVBeast", "parameters": [ "Distance" ] }
	]`))

	s.Require().NoError(err)
	s.Require().Len(got, 2)

	// Position is the whole point: a device names a model by where it sits.
	s.Require().Equal("HD2_AmpSVBeastNrm", string(got[0].ID))
	s.Require().Equal([]string{"Drive", "Bass", "Mid"}, got[0].Params)
	s.Require().Equal("HD2_Cab8x10SVBeast", string(got[1].ID))
}

func (s *SymbolsTestSuite) TestAnInstallationWithoutOneIsNotAFailure() {
	// A catalog without the table still describes what the device can do.
	// Only reading a preset off the hardware needs it.
	got, err := readSymbols(s.T().TempDir())

	s.Require().NoError(err)
	s.Require().Nil(got)
}

func (s *SymbolsTestSuite) TestReportsATableThatWillNotParse() {
	_, err := readSymbols(s.write("not json"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), symbolFile)
}

func (s *SymbolsTestSuite) TestReportsATableItCannotRead() {
	dir := s.write("[]")
	s.Require().NoError(os.Chmod(filepath.Join(dir, symbolFile), 0o000))

	_, err := readSymbols(dir)

	s.Require().Error(err)
}

func (s *SymbolsTestSuite) TestReadsTheColoursADeviceNamesItsSwitchesBy() {
	dir := s.write("[]")
	s.Require().NoError(os.WriteFile(filepath.Join(dir, controlsFile), []byte(`{
		"blend": { "format": "%.0f" },
		"footswitchLED": { "isDiscrete": true,
		  "format": ["Auto Color", "White", "Green", "Violet"] }
	}`), 0o600))

	got, err := readLEDColours(dir)

	s.Require().NoError(err)
	s.Require().Equal([]string{"auto color", "white", "green", "violet"}, got)
}

func (s *SymbolsTestSuite) TestAnInstallationWithoutColours() {
	got, err := readLEDColours(s.T().TempDir())

	s.Require().NoError(err)
	s.Require().Nil(got)
}

func (s *SymbolsTestSuite) TestReportsColoursThatWillNotParse() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, controlsFile), []byte("not json"), 0o600))

	_, err := readLEDColours(dir)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), controlsFile)
}

func (s *SymbolsTestSuite) TestReportsAColourListOfTheWrongShape() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(dir, controlsFile),
		[]byte(`{"footswitchLED": {"format": "%.0f"}}`), 0o600))

	_, err := readLEDColours(dir)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), ledControl)
}

func (s *SymbolsTestSuite) TestReportsColoursItCannotRead() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, controlsFile)
	s.Require().NoError(os.WriteFile(path, []byte("{}"), 0o600))
	s.Require().NoError(os.Chmod(path, 0o000))

	_, err := readLEDColours(dir)

	s.Require().Error(err)
}

func (s *SymbolsTestSuite) TestBuildReportsColoursItCannotRead() {
	dir := s.write("[]")
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "amp.models"), []byte("[]"), 0o600))
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, controlsFile), []byte("not json"), 0o600))

	_, err := Build(Options{ResourcesDir: dir, DeviceID: 1})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), controlsFile)
}

func (s *SymbolsTestSuite) TestBuildReportsATableItCannotRead() {
	// A catalog is generated from somebody's own installation, and a file in
	// it that cannot be read is worth complaining about rather than quietly
	// producing a catalog nothing can name a model with.
	dir := s.write("not json")
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "amp.models"), []byte("[]"), 0o600))

	_, err := Build(Options{ResourcesDir: dir, DeviceID: 1})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), symbolFile)
}

func TestSymbolsTestSuite(t *testing.T) {
	suite.Run(t, new(SymbolsTestSuite))
}
