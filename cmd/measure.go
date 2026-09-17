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

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// measureFile is the recording to measure.
var measureFile string

// measureDir is a tree of recordings to measure together.
var measureDir string

// measureCmd represents the measure command.
var measureCmd = &cobra.Command{
	Use:   "measure",
	Short: "Measure what a recording sounds like",
	Long: `Read a recording and report it as numbers.

Where the energy sits, where the sound's centre of gravity is, how sharply
notes start, how long they take to die away, how compressed the playing is,
and how much sits above the fundamental.

Nothing here decides what those numbers mean. "Warm" and "percussive" are
judgements two people disagree about; a centroid of 410Hz is not. Turning one
into the other is somebody else's job, and keeping them apart is what leaves
anywhere to stand when they disagree.

One record is one engineer's decisions on one day. Point --dir at several by
the same player and the report gains what they have in common and how much
the record chosen moved it.

WAV only. Convert anything else on the way in:

    ffmpeg -i take.mp3 take.wav
    tonestack measure --file take.wav

    just stems ~/music/mike-dirnt ~/stems
    tonestack measure --dir ~/stems/htdemucs`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if measureDir != "" {
			return measureTree(cmd, measureDir)
		}

		f, err := os.Open(measureFile) //nolint:gosec // the path is the user's own file
		if err != nil {
			return fmt.Errorf("reading %s: %w", measureFile, err)
		}

		// Opened read-only, so Close has nothing to report the read did not.
		defer func() { _ = f.Close() }()

		samples, rate, err := audio.Read(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", measureFile, err)
		}

		return cli.Profile(cmd.OutOrStdout(), audio.Measure(samples, rate))
	},
}

// measureTree measures every recording under a directory and reports them
// both one at a time and together.
func measureTree(
	cmd *cobra.Command,
	dir string,
) error {
	got, err := audio.MeasureAll(os.DirFS(dir), ".")
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	if len(got) == 0 {
		return fmt.Errorf(
			"no .wav recordings under %s: separate a directory of records with `just stems <in> <out>` first",
			dir,
		)
	}

	if err := cli.Tracks(cmd.OutOrStdout(), got); err != nil {
		return err
	}

	all := make([]audio.Profile, 0, len(got))
	for _, n := range got {
		all = append(all, n.Profile)
	}

	return cli.Across(cmd.OutOrStdout(), audio.Together(all))
}

func init() {
	rootCmd.AddCommand(measureCmd)

	measureCmd.Flags().StringVar(&measureFile, "file", "",
		"the recording to measure, as a .wav")
	measureCmd.Flags().StringVar(&measureDir, "dir", "",
		"a tree of .wav recordings to measure together")

	measureCmd.MarkFlagsOneRequired("file", "dir")
	measureCmd.MarkFlagsMutuallyExclusive("file", "dir")
}
