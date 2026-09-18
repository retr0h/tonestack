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
	"strings"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// measureFile is the recording to measure.
var measureFile string

// measureDir is a tree of recordings to measure together.
var measureDir string

// measureCorpus is a tree of players to measure and compare.
var measureCorpus string

// measureEvidence writes the measurements as rig evidence rather than as a
// report to read.
var measureEvidence bool

// measureManifest names the recordings and links them, so evidence can say
// where a figure came from.
var measureManifest string

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
    tonestack measure --dir ~/stems/htdemucs

Point --corpus at a tree holding one directory per player and the report says
which words each one's records earn. A word is earned by sitting clear of the
other players, so this is the only mode that produces any: one player has
nobody to be clear of.

Point it at one instrument. A bass centroid sits an octave below a guitar's, so
a corpus holding both would earn every bassist "dark" and every guitarist
"bright" and mean nothing by either.

    tonestack measure --corpus resources/music/bass
    tonestack measure --corpus resources/music/bass --evidence`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if measureCorpus != "" {
			return measurePlayers(cmd, measureCorpus)
		}

		if measureDir != "" {
			return measureTree(cmd, measureDir)
		}

		// Evidence is written per recording across a corpus, and one take is
		// not a corpus. Refused rather than ignored: a flag that silently does
		// nothing is how somebody concludes the feature is broken.
		if measureEvidence {
			return fmt.Errorf("--evidence reads several records: give it --dir rather than --file")
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

// measurePlayers measures every player under a tree and reports what each
// one's figures earn them against the others.
func measurePlayers(
	cmd *cobra.Command,
	dir string,
) error {
	players, err := audio.Corpus(os.DirFS(dir), ".")
	if err != nil {
		return err
	}

	if len(players) == 0 {
		return fmt.Errorf(
			"no players under %s: one directory of .wav recordings each, "+
				"separated with `just stems <in> <out>`",
			dir,
		)
	}

	// The same comparison, written two ways: a table to read, or the terms
	// it earned as evidence to paste into a rig. Both carry the figures,
	// because a word without them is an assertion.
	if measureEvidence {
		return cli.PlayerTerms(cmd.OutOrStdout(), players)
	}

	return cli.Players(cmd.OutOrStdout(), players)
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

	got, err = withSources(cmd, got)
	if err != nil {
		return err
	}

	if measureEvidence {
		return cli.Evidence(cmd.OutOrStdout(), got)
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

// withSources attaches what a manifest knows to what was measured.
//
// Without one the recordings come back untouched, and their evidence goes out
// with no link on it. That is a worse rig rather than a broken one, so it is
// allowed and said out loud rather than refused.
func withSources(
	cmd *cobra.Command,
	got []audio.Named,
) ([]audio.Named, error) {
	if measureManifest == "" {
		return got, nil
	}

	f, err := os.Open(measureManifest) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", measureManifest, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	m, err := audio.ReadManifest(f)
	if err != nil {
		return nil, err
	}

	// To stderr, never to stdout. The evidence is written to be redirected
	// into a rig, and a warning in that stream would end up inside the file.
	missing, unnamed := m.Unmatched(got)
	if len(missing) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"named in the manifest but not measured: %s\n", strings.Join(missing, ", "))
	}

	if len(unnamed) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"measured but not in the manifest, so their evidence carries no link: %s\n",
			strings.Join(unnamed, ", "))
	}

	return m.Join(got), nil
}

func init() {
	rootCmd.AddCommand(measureCmd)

	measureCmd.Flags().StringVar(&measureFile, "file", "",
		"the recording to measure, as a .wav")
	measureCmd.Flags().StringVar(&measureDir, "dir", "",
		"a tree of .wav recordings to measure together")
	measureCmd.Flags().StringVar(&measureManifest, "manifest", "",
		"a corpus manifest naming the recordings and linking them")
	measureCmd.Flags().BoolVar(&measureEvidence, "evidence", false,
		"write the measurements as rig evidence, to paste into a chain")

	measureCmd.Flags().StringVar(&measureCorpus, "corpus", "",
		"a tree of players, one directory each, compared against each other")

	measureCmd.MarkFlagsOneRequired("file", "dir", "corpus")
	measureCmd.MarkFlagsMutuallyExclusive("file", "dir", "corpus")
}
