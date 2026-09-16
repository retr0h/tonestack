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

package audio

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/go-audio/wav"
)

// Reading a file into the samples a measurement works on.
//
// One format, WAV, because it is the one a decoder can be trusted with in a
// few lines. Anything else is converted on the way in by whoever has ffmpeg,
// which is the same bargain the gear map strikes with Python: shell out for
// the one job Go should not be doing, and keep the part that matters here.

// ErrNotAudio reports a file that is not audio this can read.
var ErrNotAudio = errors.New("not readable audio")

// NotAudioError says what was wrong with it.
type NotAudioError struct {
	// Why names the thing that was wrong.
	Why string
}

// Error implements the error interface.
func (e *NotAudioError) Error() string {
	return fmt.Sprintf("not readable audio: %s", e.Why)
}

// Unwrap returns ErrNotAudio so callers can match with errors.Is.
func (*NotAudioError) Unwrap() error { return ErrNotAudio }

// Read decodes a WAV into single-channel samples between -1 and 1.
//
// Several channels are averaged into one. A measurement describes a sound
// rather than a stereo image, and a bass part that sits in the middle of a mix
// measures the same either way.
//
// The reader must also seek, which a file does and a pipe does not. Line 6's
// own format needs the same, for the same reason.
func Read(
	r io.ReadSeeker,
) ([]float64, int, error) {
	dec := wav.NewDecoder(r)

	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, 0, &NotAudioError{Why: err.Error()}
	}

	// A header this cannot make sense of is refused above rather than here: a
	// file naming no channels fails as "format not supported", and one naming
	// no bit depth as "unhandled byte depth". Both arrive as that error, so
	// anything reaching this point has a format, a rate and a width.
	rate := buf.Format.SampleRate
	channels := buf.Format.NumChannels

	// What divides a whole sample down to something between -1 and 1. A
	// decoder hands back whatever width the file was written at, and a
	// measurement that skipped this would report a 24-bit recording as
	// hundreds of times louder than the same take at 16.
	full := math.Pow(2, float64(dec.BitDepth-1))

	out := make([]float64, 0, len(buf.Data)/channels)

	for at := 0; at+channels <= len(buf.Data); at += channels {
		var sum float64
		for c := range channels {
			sum += float64(buf.Data[at+c]) / full
		}

		out = append(out, sum/float64(channels))
	}

	return out, rate, nil
}
