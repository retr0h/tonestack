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

package chain

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// ValidateTopology reports a rig whose shape the device cannot represent:
// too many blocks, a processor that does not exist, or positions on one
// processor that are not the contiguous run 0..n-1.
//
// It needs no catalog — every question it answers is about the rig alone.
func ValidateTopology(s Chain, lim Limits) error {
	if len(s.Blocks) == 0 {
		return &TopologyError{Reason: "rig has no blocks"}
	}

	if len(s.Blocks) > lim.MaxBlocks {
		return &TopologyError{
			Reason: fmt.Sprintf(
				"%d blocks, device holds %d",
				len(s.Blocks),
				lim.MaxBlocks,
			),
		}
	}

	byChip := make(map[int][]int)

	for _, b := range s.Blocks {
		if b.DSP < 0 || b.DSP >= lim.Paths {
			return &TopologyError{
				Reason: fmt.Sprintf(
					"block %q is on processor %d, device has %d",
					b.Model, b.DSP, lim.Paths,
				),
			}
		}

		if b.Pos < 0 {
			return &TopologyError{
				Reason: fmt.Sprintf(
					"block %q has negative position %d",
					b.Model,
					b.Pos,
				),
			}
		}

		byChip[b.DSP] = append(byChip[b.DSP], b.Pos)
	}

	for _, chip := range slices.Sorted(maps.Keys(byChip)) {
		positions := byChip[chip]
		slices.Sort(positions)

		for i, p := range positions {
			if p != i {
				return &TopologyError{
					Reason: fmt.Sprintf(
						"processor %d positions are not contiguous from zero: %v",
						chip,
						positions,
					),
				}
			}
		}
	}

	return validateSnapshots(s)
}

func validateSnapshots(s Chain) error {
	if len(s.Snapshots) == 0 {
		return nil
	}

	for _, snap := range s.Snapshots {
		for _, key := range slices.Sorted(maps.Keys(snap.Overrides)) {
			// JSON object keys are strings, so a block index arrives as "0".
			idx, err := strconv.Atoi(key)
			if err != nil {
				return &TopologyError{
					Reason: fmt.Sprintf(
						"snapshot %q overrides block %q, which is not an index",
						snap.Name, key,
					),
				}
			}

			if idx < 0 || idx >= len(s.Blocks) {
				return &TopologyError{
					Reason: fmt.Sprintf(
						"snapshot %q overrides block %d, rig has %d",
						snap.Name, idx, len(s.Blocks),
					),
				}
			}
		}
	}

	return nil
}
