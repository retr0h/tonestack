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

package cmd

import (
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// debugEnv turns on the wire trace: every USB frame in and out, on stderr.
const debugEnv = "TONESTACK_USB_DEBUG"

// deviceUsage is what --device says, wherever it is offered.
//
// One string, because a flag that means the same thing in nine places should
// read the same way in all of them.
const deviceUsage = "which pedal's built-in catalog to use: " +
	"HX Stomp, HX Stomp XL, Helix Floor or Helix LT"

// newClient builds the Client a command calls.
//
// The library reads none of this tool's environment, so it is read here and
// passed in: sdk.DumpEnv names a file each device answer is written to, and
// debugEnv traces every frame to stderr.
func newClient(
	opts ...sdk.Option,
) *sdk.Client {
	if path := os.Getenv(sdk.DumpEnv); path != "" {
		opts = append(opts, sdk.WithCapture(dumpFile(path)))
	}

	if os.Getenv(debugEnv) != "" {
		opts = append(opts, sdk.WithTrace(os.Stderr))
	}

	return sdk.New(opts...)
}

// clientFlags are the flags that describe a Client rather than a call.
//
// Empty means the default in every case: the built-in catalog and statistics,
// the rigs that ship, and the state directory for backups.
type clientFlags struct {
	catalog   string
	device    string
	stats     string
	recipes   string
	backupDir string
}

// client builds the Client these flags describe, and whatever else opts say.
func (f *clientFlags) client(
	opts ...sdk.Option,
) *sdk.Client {
	return newClient(append([]sdk.Option{
		sdk.WithCatalog(f.catalog),
		sdk.WithDevice(f.device),
		sdk.WithStats(f.stats),
		sdk.WithRecipes(f.recipes),
		sdk.WithBackupDir(f.backupDir),
	}, opts...)...)
}

// dumpFile is where a device's answer is kept.
//
// Each answer replaces the last, which is what the variable has always meant:
// a command reads one slot, and the file holds that slot.
type dumpFile string

// Write puts p in the file, replacing whatever it held.
func (d dumpFile) Write(
	p []byte,
) (int, error) {
	if err := os.WriteFile(string(d), p, 0o600); err != nil {
		return 0, fmt.Errorf("writing %s: %w", string(d), err)
	}

	return len(p), nil
}
