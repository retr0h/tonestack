# Talking to a device

Two ways to reach an HX Stomp, and the reason only one of them carries presets
in both directions.

## Reading is live; writing is not

`presets list`, `show` and `export` talk to the device over USB. Nothing is
selected, loaded or written. The device answers and goes on playing whatever it
was:

```bash
tonestack presets list
tonestack presets show   --slot 31A
tonestack presets export --slot 31A --out lead.yaml
```

`copy` and `swap` write to the device as well. They move a preset from one slot
to another exactly as the device wrote it: nothing is decoded and nothing is
rebuilt, which is what makes them the safest thing to write. A preset is seeked
through by a table of byte offsets, and the surest way to keep those right is to
change nothing.

The destination is overwritten, and a device has no undo.

Building a preset from a rig and putting it on the device still goes through a
file HX Edit wrote:

```text
HX Edit  ──backup──▶  device.hlb  ──▶  tonestack  ──▶  edited.hlb  ──restore──▶  HX Edit
```

`copy`, `swap` and `import` work on that file, and every reading command takes
`--file` too, for working from a backup with no device attached. A `.hlb` holds
every setlist, so one backup is the whole instrument and one restore puts it
back.

An earlier version of this document claimed writing over USB was unsolved. That
was wrong, and researching it properly is what produced
[protocol.md](protocol.md): two independent projects write presets over USB and
have done for months. What remains true is narrower and still matters.

**What the device gives back is not a `.hlx`.** It is an internal MessagePack
document beginning with a 48-byte table of byte offsets into itself. The device
seeks with that table rather than walking the document, so a re-encode that
changes any field's byte width shifts every offset after it. The device accepts
such a write and then reads the preset as empty. Both projects lost hardware
sessions to exactly this.

## Editing a section means splicing it

Decoding a section, changing it and encoding it again does not work. Line 6
choose an encoding per field rather than per value, and nothing recovers that
choice from the value afterwards. Across the three captured presets a number
that would fit a one-byte fixint is written as `uint8` 282 times and as `uint32`
ten times, and every one of the 140 floats is a `float32` where Go's encoder
writes `float64`.

Decoding section 0 of `preset.bin` and encoding it again returns 835 bytes where
the device wrote 588. Whole documents grow by around 40%:

| section         | preset.bin  | switches.bin | empty.bin   |
| --------------- | ----------- | ------------ | ----------- |
| 0, blocks       | 588 → 835   | 567 → 773    | 248 → 369   |
| 3, footswitches | 136 → 187   | 209 → 284    | 10 → 13     |
| 10              | 1460 → 1950 | 1520 → 2061  | 1520 → 2061 |

`wire.Locate` and `wire.Splice` exist because of that. A path names one value,
`Locate` returns the bytes it occupies, and `Splice` swaps those bytes for new
ones. Everything outside that range is copied through unread, so it survives
whatever the device chose for it. `Document.Encode` then rebuilds the offset
table over the new lengths.

A path is a list of integers, because every map key in a preset is one. There
are 665 across the captures and not a single string. The container decides
whether a step reads as a map key or as an array index. A block's model number
is `{22, i, 20, 24, 25}`, whether it is on is `{22, i, 20, 10}`, and its
parameters are `{22, i, 20, 11, 4, j}`.

A replacement keeps the width the device wrote wherever the new value still fits
it, so swapping one parameter for another leaves the section the length it was.
A value needing more room widens to the narrowest form that holds it.

`TestSpliceRawKeepsEveryOtherByte` locates all 5,413 values in all three
captures and writes each one back as itself, asserting the section is unchanged
every time.

**Writing a preset synthesised from nothing is the least-solved thing in the
space, and neither project does it.** The reliable shape is to read a preset off
the device, change it, and write it back. That is also what the corpus says from
a different direction: 98.6% of real presets carry `inputA`, `outputA`, `split`
and `join` routing that a generated one has none of.

## What USB is for

`pkg/sdk` reaches the hardware over USB for what a file cannot answer: which
devices are attached, what each slot holds, and what one slot actually contains.

It is the only package needing cgo, which is why it is the only one that cannot
be cross-compiled or built with `CGO_ENABLED=0`.

The import graph decides cgo, not the module boundary, so nothing that avoids
importing `pkg/sdk` pays for it. A future HTTP service never touches a device
and stays pure Go.

## Prerequisites

```bash
brew install libusb        # macOS
apt-get install libusb-1.0-0-dev   # Debian/Ubuntu
```

`github.com/google/gousb` binds to it. macOS needs no special privileges,
because enumeration reads descriptors only and never opens a device.

## Two identifier systems, deliberately separate

A device answers to a **USB product ID** on the bus and is named by a different
**device ID** inside a preset. They are unrelated numbers and both are needed.

| Device      | USB `vid:pid` |
| ----------- | ------------- |
| HX Stomp    | `0e41:4246`   |
| HX Stomp XL | `0e41:4253`   |
| Helix Floor | `0e41:4248`   |
| Helix LT    | `0e41:424a`   |

The other identifier is the integer a preset carries in `data.device`, listed in
[preset-format.md](preset-format.md). `sdk.Models` holds both, because resolving
one to the other is exactly this package's job.

`sdk.Models` holds the mapping. Product IDs were observed on the bus; device IDs
come from the preset corpus. A device missing from that table is still reachable
over USB but will not be named, and presets cannot be written for it.

## Hardware stays behind an interface

Finding a device, choosing between two, claiming an interface and waiting on a
busy one all run against interfaces this package declares, so a test supplies
them and no device is attached:

```go
type Bus interface {
	Devices(match func(vendor, product uint16) bool) ([]handle, error)
	Close() error
}
```

`pkg/sdk/usb.go` is every call this project makes into libusb, one expression
per method. Keep it that way. Anything holding a decision belongs on the other
side of an interface where a test can reach it, and if that file grows past
forwarding then logic has leaked into the half nothing checks.

The file is counted in the coverage total rather than excluded from it. An
exclusion hides how big a file is; the 99% target says what cannot be reached
and gets worse if that file grows.

## Builds without cgo still work

`usb.go` carries `//go:build cgo`; `usb_nocgo.go` provides the same surface for
builds without it, returning `ErrNoUSBSupport` from every call.

That keeps `go install` working for someone who has no libusb. Describing a
chain, validating it and writing a preset are all pure Go, so they get
everything except device access, and a clear message rather than a link error if
they try to reach hardware.

CI installs libusb so `pkg/sdk` is compiled, vetted and linted like everything
else. Running CI with `CGO_ENABLED=0` would avoid the system dependency but
would leave that package unchecked anywhere except a developer's machine.
