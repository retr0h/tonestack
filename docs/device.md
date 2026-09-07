# Talking to a device

Reaching an HX Stomp over USB, and what a preset holds that nothing here can
generate.

## What is live

`presets list`, `show` and `export` talk to the device over USB. Nothing is
selected, loaded or written. The device answers and goes on playing whatever it
was:

```bash
tonestack presets list
tonestack presets show   --slot 31A
tonestack presets export --slot 31A --out lead.yaml
```

## Writing works

`copy`, `swap` and `import` write the attached device. Verified on an HX Stomp
on 7 September 2026: a preset copied into an empty slot, two slots exchanged,
and a preset built from a recipe written into a third and read back with the
gear it was asked for.

Two things had to be right, and each produced a different failure.

**A document goes out under the tag a device uses.** A device sends a preset
under MessagePack's string tag, `str16`, and takes one back under the same tag.
A generic encoder picks the narrowest binary tag that fits, `bin16`. The bytes
are identical and the tag is not, and the device answers `error -3`, a reference
it does not recognise. The same code answers `error -3` for a setlist that does
not exist, which is what that code means.

**A slot write is finished when it is answered.** The erase and program that
follow never appear on the wire. Waiting for a completion notification waits for
one that is not coming, for ten seconds, while the preset already sits in the
slot. Both status 0 and status 1 have been seen for a write that landed, so
neither is read.

What is left is the pause. Nothing on the wire says when the flash finishes, so
a second write landing on the first stacks its commit, and 750ms between them is
the only thing keeping them apart.

An earlier version of this document said writing was implemented and had never
worked. It was, and the two reasons are above.

## A session has to be closed

Every command opens three channels and, when it is done, tells the device on
each of them that the session is over. The message that opens a channel is the
one that closes it: it is a session boundary and appears at both ends of the
conversation.

Leaving it out does not fail. The command works, the device answers, and the
pedal is left believing an editor is still attached: turn the dial and the
footswitches stop changing with the preset, because the front panel waits for an
editor to tell it what to show. Confirmed on hardware in both directions.

## Selecting a preset has to be waited for

`presets select` loads a preset, which is what stepping on a footswitch does.
Nothing is written and the slot it came from is untouched.

A select is deferred. The device says it has taken the request and finishes
afterwards, and it answers other questions while the switch is still in flight,
so "it answered again" is not "it finished". Returning early and closing the
session leaves the device holding a half-finished switch, and it settles that by
wiping its edit buffer: the preset comes up with no blocks and no footswitch
colours.

Asking the device what it is playing, until it names the preset that was asked
for, is the only honest signal. Confirmed on hardware in both directions.
Without the wait the colours went; with it they stayed.

## Reading has a cutoff nobody has explained

Reading a preset stopped working for the higher slots during one session, and a
power cycle did not bring it back while HX Edit read the same device fine.

An earlier version of this section put the boundary at slot 29 and was wrong:
the device it was measured on holds nothing at all between slots 28 and 77, and
a slot holding nothing answers with nothing whichever side of a boundary it sits
on. What is actually known is narrower.

| slot  | index | holds    | answer       |
| ----- | ----- | -------- | ------------ |
| `01A` | 0     | a preset | the document |
| `09A` | 24    | a preset | the document |
| `10A` | 27    | a preset | the document |
| `27A` | 78    | a preset | nothing      |
| `31A` | 90    | a preset | nothing      |
| `34A` | 99    | a preset | nothing      |

Slots 0 to 27 answer and 78 upwards do not, with no preset in between to narrow
it with. `27B` answered with its whole document earlier the same day, and the
device still loads all of them when asked to: `presets select` on `34A` plays
it. The preset list, a different opcode on a different channel, names them
throughout.

Unexplained. It is recorded because a sharp reproducible boundary is worth more
than the theories that did not survive: it is not the channel, not the missing
`101: 2` argument that opcode 4 is documented to take, and not a session
bootstrap.

## Working from a file

Every command takes `--file`, for working from a backup with no device attached.
A `.hlb` holds every setlist, so one backup is the whole instrument and one
restore puts it back:

```text
HX Edit  ──backup──▶  device.hlb  ──▶  tonestack  ──▶  edited.hlb  ──restore──▶  HX Edit
```

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

## A chain goes into a preset a device wrote

A device lays a chain out on a fixed grid of 20 positions. The first holds the
input, the tenth and eleventh the split and the join, the last the output, and
the device decides where those sit. The other 16 hold blocks or nothing.

`wire.Place` writes every position that can hold a block. One the chain names
gets that block and one it does not gets emptied, so a preset says the same
thing whatever it held before. A block entry reads:

```text
19: 6                                    a block
20:
  9:  1 | 15 | 17 | 18                   what it is
  10: true                               switched on
  11: {2: count, 3: named, 4: [values]}  its parameters
  12: {2: count, 3: named, 4: [values]}  the cabinet it carries, if any
  24: {23: carries one, 25: model, 26: cabinet model}
```

Key `9` is read off the captures rather than documented. Every effect says 1, a
cabinet on its own says 15, an amp with no cabinet says 17, and an amp carrying
one says 18. That is ten blocks across three presets, so treat it as a rule that
has not been contradicted rather than one anybody confirmed.

### The snapshots are part of the chain

Each of the three snapshots carries an array of 20 in step with the grid,
holding whether each position is switched on. Snapshot 0 of the bass capture
matches the live chain exactly, and the other two differ, which is what a
snapshot is for.

So a chain written without them recalls the wrong blocks the moment anybody
presses a snapshot. `Place` writes both or neither.

### Every float is a float32

A device stores parameters as float32. Writing 0.45 and reading it back gives
0.44999998807907104. A rig lifted off hardware already carries values that have
been through this and survives it unchanged; one somebody typed by hand gets
rounded to what the device can store.

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
