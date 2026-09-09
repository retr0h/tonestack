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

// Package wire is what a Helix device says over its editor endpoint: the
// framing, and the documents the framing carries.
//
// Two halves, and the second is the larger one. Framing is a frame, an
// envelope and a request or reply — wire.go, frame.go, rpc.go, encode.go,
// decode.go. The rest is the preset document a device sends and takes back:
// skimming MessagePack without decoding it, splicing a byte range in place,
// and placing a chain on the grid. Both are what the device speaks, so they
// live together, and it is worth knowing which half a change is in.
//
// Pure Go, and deliberately separate from the USB transport in pkg/sdk: none
// of it has hardware in it, so it can be exercised in full without a device
// attached. The transport is the untestable half and is kept thin.
//
// The format is not published by Line 6. It was reverse engineered by
// tonepush and fretwire, both MIT licensed, and their documentation is what
// this package implements. See docs/protocol.md.
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// EnvelopeSize is the fixed prefix on every envelope.
const EnvelopeSize = 8

// MaxBody is the largest body this package will decode.
//
// A frame arrives over a 512-byte bulk endpoint and a preset document runs to
// tens of kilobytes, so the ceiling exists to bound a corrupted length field
// rather than to reflect a limit the device has.
const MaxBody = 1 << 20

// Originator says which end of the link sent a frame.
//
// Host frames always carry 1 and device frames always 0, without exception,
// which makes it the cheapest check that a stream is still aligned.
type Originator uint16

// The two originators.
const (
	FromDevice Originator = 0
	FromHost   Originator = 1
)

// Envelope is the innermost header, wrapping the MessagePack body.
type Envelope struct {
	// Originator is who sent it.
	Originator Originator
	// Service selects which conversation the frame belongs to. Frames from
	// the device may carry a value that means nothing.
	Service uint16
	// Body is the MessagePack payload.
	Body []byte
}

// Sentinels callers match with errors.Is.
var (
	// ErrShortFrame reports a frame that ended before its header did.
	ErrShortFrame = errors.New("frame is too short")
	// ErrBodyTooLarge reports a length field no real frame would carry.
	ErrBodyTooLarge = errors.New("frame body is too large")
)

// BodyTooLargeError names the length that was refused.
type BodyTooLargeError struct {
	Length uint32
}

func (e *BodyTooLargeError) Error() string {
	return fmt.Sprintf(
		"frame body is too large: %d bytes, limit is %d", e.Length, MaxBody)
}

func (*BodyTooLargeError) Unwrap() error { return ErrBodyTooLarge }

// EncodeEnvelope renders an envelope.
//
//	offset  size  field
//	0       2     originator
//	2       2     service
//	4       4     body length, little endian
//	8       n     body
func EncodeEnvelope(f Envelope) []byte {
	out := make([]byte, EnvelopeSize+len(f.Body))

	binary.LittleEndian.PutUint16(out[0:2], uint16(f.Originator))
	binary.LittleEndian.PutUint16(out[2:4], f.Service)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(f.Body)))
	copy(out[EnvelopeSize:], f.Body)

	return out
}

// DecodeEnvelope reads one envelope from raw bytes, returning it and what follows.
//
// A bulk read can carry more than one frame, so the remainder is handed back
// rather than discarded.
func DecodeEnvelope(raw []byte) (Envelope, []byte, error) {
	if len(raw) < EnvelopeSize {
		return Envelope{}, nil, fmt.Errorf(
			"%w: %d bytes, need at least %d", ErrShortFrame, len(raw), EnvelopeSize)
	}

	n := binary.LittleEndian.Uint32(raw[4:8])
	if n > MaxBody {
		return Envelope{}, nil, &BodyTooLargeError{Length: n}
	}

	end := EnvelopeSize + int(n)
	if len(raw) < end {
		return Envelope{}, nil, fmt.Errorf(
			"%w: body is %d bytes, only %d arrived", ErrShortFrame, n, len(raw)-EnvelopeSize)
	}

	return Envelope{
		Originator: Originator(binary.LittleEndian.Uint16(raw[0:2])),
		Service:    binary.LittleEndian.Uint16(raw[2:4]),
		Body:       raw[EnvelopeSize:end],
	}, raw[end:], nil
}

// ReadEnvelope decodes one envelope from a stream.
func ReadEnvelope(r io.Reader) (Envelope, error) {
	head := make([]byte, EnvelopeSize)
	if _, err := io.ReadFull(r, head); err != nil {
		return Envelope{}, fmt.Errorf("reading frame header: %w", err)
	}

	n := binary.LittleEndian.Uint32(head[4:8])
	if n > MaxBody {
		return Envelope{}, &BodyTooLargeError{Length: n}
	}

	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return Envelope{}, fmt.Errorf("reading frame body: %w", err)
	}

	return Envelope{
		Originator: Originator(binary.LittleEndian.Uint16(head[0:2])),
		Service:    binary.LittleEndian.Uint16(head[2:4]),
		Body:       body,
	}, nil
}
