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

package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Sizes of the two headers that wrap an envelope.
const (
	// FrameSize is the outermost header, one per USB transfer.
	FrameSize = 8
	// ChannelSize is the header carrying sequence and acknowledgement.
	ChannelSize = 8
)

// Frame flags.
const (
	// FlagNormal is every frame but the channel opening.
	FlagNormal uint8 = 0x18
	// FlagHandshake marks a channel being opened.
	FlagHandshake uint8 = 0x28
)

// Message types.
//
// This is a bit field rather than an enumeration, which matters: a frame
// carrying data may also piggyback an acknowledgement, so a client testing
// for equality with MsgData silently drops everything that arrives as
// MsgData|MsgAck.
const (
	// MsgHello opens a channel, and with FlagNormal and no body closes one.
	MsgHello uint16 = 0x0002
	// MsgData means the frame carries stream bytes.
	MsgData uint16 = 0x0004
	// MsgAck is a bare acknowledgement.
	MsgAck uint16 = 0x0008
	// MsgKeepAlive holds a channel open.
	MsgKeepAlive uint16 = 0x0010
)

// AckBase is what an acknowledgement counts up from.
//
// Not zero, and not decoration: a client sending a bare count of bytes
// received is ignored by the device.
const AckBase uint32 = 0x1000

// Frame is one message on the editor endpoint.
//
// A single USB transfer can hold several back to back, so a reader has to
// decode all of them rather than the first.
type Frame struct {
	// Flags is FlagNormal or FlagHandshake.
	Flags uint8
	// DeviceNode and HostNode identify the channel. Host frames put the
	// device first; the device puts them the other way round.
	DeviceNode uint16
	HostNode   uint16
	// Seq counts frames sent on this channel. Big endian on the wire.
	Seq uint16
	// Type is a bit field of the Msg constants. Big endian on the wire.
	Type uint16
	// Ack is AckBase plus the stream bytes consumed on this channel.
	Ack uint32
	// Payload is whatever follows the two headers: envelope bytes for a data
	// frame, a fixed tail for a handshake, nothing for an acknowledgement.
	Payload []byte
}

// ErrShortTransfer reports a transfer that ended before its headers did.
var ErrShortTransfer = errors.New("transfer is too short")

// EncodeFrame renders a frame, padded to a four-byte boundary.
//
//	frame    [0..3) length u24 LE, [3] flags, [4..6) device, [6..8) host
//	channel  [0..2) seq u16 BE, [2..4) type u16 BE, [4..8) ack u32 LE
//
// The endianness is genuinely mixed. Sequence and type are big endian and
// everything else is little endian, which is invisible while the values are
// small because the high bytes are zero either way.
func EncodeFrame(f Frame) []byte {
	body := ChannelSize + len(f.Payload)
	out := make([]byte, FrameSize+body, FrameSize+pad4(body))

	out[0] = byte(body)
	out[1] = byte(body >> 8)
	out[2] = byte(body >> 16)
	out[3] = f.Flags

	binary.LittleEndian.PutUint16(out[4:6], f.DeviceNode)
	binary.LittleEndian.PutUint16(out[6:8], f.HostNode)

	binary.BigEndian.PutUint16(out[8:10], f.Seq)
	binary.BigEndian.PutUint16(out[10:12], f.Type)
	binary.LittleEndian.PutUint32(out[12:16], f.Ack)

	copy(out[FrameSize+ChannelSize:], f.Payload)

	return out[:cap(out)]
}

// DecodeFrame reads one frame, returning it and what follows in the transfer.
//
// Padding to a four-byte boundary is skipped rather than returned. The device
// does not always zero it, so it cannot be treated as the start of anything.
func DecodeFrame(raw []byte) (Frame, []byte, error) {
	if len(raw) < FrameSize+ChannelSize {
		return Frame{}, nil, fmt.Errorf(
			"%w: %d bytes, need at least %d",
			ErrShortTransfer, len(raw), FrameSize+ChannelSize)
	}

	body := int(raw[0]) | int(raw[1])<<8 | int(raw[2])<<16
	if body < ChannelSize {
		return Frame{}, nil, fmt.Errorf(
			"%w: frame declares %d bytes, less than a channel header",
			ErrShortTransfer, body)
	}

	end := FrameSize + body
	if len(raw) < end {
		return Frame{}, nil, fmt.Errorf(
			"%w: frame declares %d bytes, %d arrived",
			ErrShortTransfer, body, len(raw)-FrameSize)
	}

	f := Frame{
		Flags:      raw[3],
		DeviceNode: binary.LittleEndian.Uint16(raw[4:6]),
		HostNode:   binary.LittleEndian.Uint16(raw[6:8]),
		Seq:        binary.BigEndian.Uint16(raw[8:10]),
		Type:       binary.BigEndian.Uint16(raw[10:12]),
		Ack:        binary.LittleEndian.Uint32(raw[12:16]),
		Payload:    raw[FrameSize+ChannelSize : end],
	}

	// Padding may hold junk, so anything shorter than a header is the end.
	next := pad4(end)
	if next > len(raw) {
		next = len(raw)
	}

	return f, raw[next:], nil
}

// CarriesData reports whether a frame holds stream bytes.
//
// Tested as a bit because the device sets it alongside an acknowledgement and
// alongside a keep-alive. Comparing for equality drops those.
func (f Frame) CarriesData() bool { return f.Type&MsgData != 0 }

// pad4 rounds a length up to a four-byte boundary.
func pad4(n int) int { return (n + 3) &^ 3 }
