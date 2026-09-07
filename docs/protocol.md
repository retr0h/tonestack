# The device protocol

How to talk to a Helix over USB, and how much of it is actually known.

Line 6 publishes nothing about this. Everything here was reverse engineered by
other people, verified against real hardware where this document says so, and
implemented in `pkg/sdk`.

## Where this came from

| Source                                                                         | What it gives                                                                                                                                            |
| ------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [tonepush](https://github.com/crmne/tonepush) (Rust, MIT)                      | [PROTOCOL.md](https://github.com/crmne/tonepush/blob/main/PROTOCOL.md), 1071 lines, every claim marked confirmed, inferred or open. The definitive text. |
| [fretwire](https://github.com/john-baxter-dev/fretwire) (Rust, MIT/Apache-2.0) | Independent corroboration, and the widest device coverage: HX Stomp, Helix Floor and POD Go all verified.                                                |
| [openhx](https://github.com/allansomensi/openhx) (Rust, MIT)                   | A clean sequential spec, captured off an HX Stomp XL.                                                                                                    |
| [helix_usb](https://github.com/kempline/helix_usb) (Python)                    | The earliest effort. Found the three-channel structure. Read-only.                                                                                       |
| `sound/usb/format.c` in the Linux kernel                                       | The authoritative product ID table.                                                                                                                      |

Nothing here is copied from those projects. They are documentation this
implementation was written against, the same way `docs/preset-format.md`
documents a format nobody published.

### Which of this came from where

Worth keeping straight, because the two kinds of claim need re-checking in
different ways. A borrowed one is re-checked by reading the source again; a
measured one by capturing another preset.

| Claim                                             | Where from                                    |
| ------------------------------------------------- | --------------------------------------------- |
| Framing, channels, the handshake, opcodes 1 and 4 | tonepush PROTOCOL.md, verified on an HX Stomp |
| Write opcodes 5 and 8, chunking, deferred commit  | tonepush PROTOCOL.md, marked confirmed there  |
| The rules that keep a device alive                | tonepush and fretwire, learned the hard way   |
| What the twelve offsets point at                  | measured here, off three captured presets     |
| Model numbers indexing `Helix.sym`                | measured here, confirmed against the corpus   |
| Footswitch label, colour and block keys           | measured here, confirmed against HX Edit      |
| The cabinet an amplifier carries, and `@type`     | measured here, confirmed against the corpus   |

Captures live in `pkg/sdk/wire/testdata`. Three slots off an HX Stomp on
firmware 3.71, which is what every measurement above was taken from.

## It is not MIDI

Worth stating plainly, because it is the first place anybody looks.

Line 6's Helix product manager: *"Helix doesn't really do SysEx."* Their
knowledge base: *"Does Helix pass SysEx data via MIDI Thru? No. We actively
filter out SysEx data through Helix."* The official MIDI documentation has no
SysEx section at all: program change, control change and clock, nothing more.

The one thing MIDI answers is a Universal Identity Request, which returns the
model and firmware revision and no preset data whatsoever.

So MIDI switches presets. It cannot read or write one. The editor protocol is
somewhere else entirely.

## The editor lives on a vendor-specific interface

Verified on an HX Stomp, firmware 3.80, by reading its descriptors:

| Interface | Class                      | Endpoints                             | Role                   |
| --------- | -------------------------- | ------------------------------------- | ---------------------- |
| **0**     | `ff/00/00` vendor-specific | bulk `0x01` OUT, `0x81` IN, 512 bytes | **the editor channel** |
| 1–3       | `01/01`, `01/02` audio     | isochronous                           | USB audio              |
| 4         | `01/03` MIDI streaming     | bulk `0x02`, `0x82`                   | ordinary musical MIDI  |
| 5         | `03/00` HID                | interrupt `0x84`, 8 bytes             | switches and knobs     |

Interface 0 has no kernel driver bound to it. **HX Edit claims it exclusively
while running**, so it has to be quit before anything else can connect.

This also settles a portability question: because the channel is raw USB rather
than MIDI, iOS cannot reach it at all. Linux, macOS, Windows and Android can.

## Product identifiers

From the kernel's sample-rate quirk table, which is the only public list.

| Product      | USB `vid:pid`                                       |
| ------------ | --------------------------------------------------- |
| Helix Floor  | `0e41:4241` before firmware 2.82, `0e41:4248` after |
| Helix Rack   | `0e41:4242`, `0e41:4249`                            |
| Helix LT     | `0e41:4244`, `0e41:424a`                            |
| HX Effects   | `0e41:4245`                                         |
| **HX Stomp** | `0e41:4246`                                         |
| POD Go       | `0e41:4247`, `0e41:424b`                            |
| HX Stomp XL  | `0e41:4253`                                         |

These are unrelated to the integer a preset carries in `data.device`. Both are
needed, and [device.md](device.md) explains why.

## Framing

Implemented in `pkg/sdk/wire`, which is pure Go and tested without hardware.

There are **three** nested headers, not one. Flattening them works by accident
because 8 + 8 = 16, and then falls apart the moment a reply is longer than one
frame.

```text
┌─ one USB bulk transfer, which may hold SEVERAL frames ──────────────┐
│ [0..3) payload length, u24 LE   [3] flags: 0x18 normal, 0x28 hello  │  frame
│ [4..6) device node, u16 LE      [6..8) host node, u16 LE            │
│ ┌─ payload ───────────────────────────────────────────────────────┐ │
│ │ [0..2) seq, u16 BIG-endian    [2..4) type, u16 BIG-endian        │ │  channel
│ │ [4..8) ack, u32 little-endian                                   │ │
│ │ ┌─ stream bytes, one message may straddle frames ─────────────┐ │ │
│ │ │ [0..2) originator, u16 LE: 1 host, 0 device                 │ │ │  envelope
│ │ │ [2..4) service id, u16 LE, GARBAGE on some device replies   │ │ │
│ │ │ [4..8) body length, u32 LE                                  │ │ │
│ │ │ [8..]  MessagePack body                                     │ │ │
│ └─└─────────────────────────────────────────────────────────────┘─┘ │
│ padding to a 4-byte boundary; device padding is not always zero      │
└─────────────────────────────────────────────────────────────────────┘
```

**The endianness is genuinely mixed.** `seq` and `type` are big-endian and
everything else is little-endian. Getting it wrong is invisible while the values
are small, because the high bytes are zero either way.

Host frames always carry originator 1 and device frames always 0, with no
exceptions, which makes it the cheapest check that a stream is still aligned.
The *service* field beside it must be ignored on device replies. The same reply
arrives carrying different values in different sessions, because it is
uninitialised memory.

`type` is a **bit field, not an enumeration**: `0x02` hello, `0x04` carries
data, `0x08` acknowledgement, `0x10` keep-alive. A frame can carry data *and*
piggyback an acknowledgement, so a client testing `type == 0x04` silently drops
everything that arrives as `0x0c`.

A single bulk read can carry more than one frame, so a decoder has to return
what follows rather than discard it.

### Channels, sequence and acknowledgement

Three conversations, each with its own counters:

| Channel | device node | host node | services  | Carries                    |
| ------- | ----------- | --------- | --------- | -------------------------- |
| control | `0x1001`    | `0x03ef`  | 5, then 2 | setlists, the preset list  |
| events  | `0x1002`    | `0x03f0`  | 4         | unsolicited notifications  |
| data    | `0x1080`    | `0x03ed`  | 6         | the current preset, params |

`seq` advances on **every** frame the host sends on that channel,
acknowledgements and keep-alives included. It starts at 0 for the opening frame
and then jumps to **2, not 1**; the device stops answering a client that sends
1\.

`ack` is `0x1000` plus the count of stream bytes consumed on that channel, where
a stream byte is the envelope including its own header. Sending a bare count
instead makes the device stop responding.

## Remote calls

Three shapes, all MessagePack maps with integer keys:

```text
request       {102: txn, 100: opcode, 101: args}
response      {102: txn, 103: status, 104: result}
notification  {105: event, 106: args}
```

Transaction identifiers start at 1000 per channel and increment.

Status, under key 103:

| Value | Meaning                                                                                            |
| ----- | -------------------------------------------------------------------------------------------------- |
| `0`   | done                                                                                               |
| `1`   | **accepted, and the operation completes later**, matched by transaction id in a later notification |
| `255` | refused, with `{111: negative error code}` in the result                                           |

A client that reads any non-zero status as failure decides every deferred
operation failed. Status `1` is not an error, and it is also not validation:
selecting preset 999 on a device holding 126 answers `1` and does nothing.

## What is known to work

| Capability                                  | State                                     |
| ------------------------------------------- | ----------------------------------------- |
| Enumerate and identify a device             | implemented, verified on hardware         |
| Framing                                     | implemented, tested against real captures |
| Session handshake                           | implemented, verified on hardware         |
| List presets (opcode 1)                     | implemented, verified on hardware         |
| Read a preset without loading it (opcode 4) | implemented, verified on hardware         |
| Write a preset (opcodes 5, 8)               | implemented, never sent to hardware       |
| Save from the edit buffer (opcode 71)       | not implemented                           |

Verified means an HX Stomp on firmware 3.80 answered, not that a test asserts
it. `pkg/sdk` needs hardware and is excluded from the coverage gate;
`pkg/sdk/wire` needs none and is covered in full.

## Reading one preset

Opcode 4, on the control channel:

```text
request   {102: txn, 100: 4, 101: {107: setlist, 108: slot}}
reply     the preset, as the three values described below
```

Read-only in the strongest sense: the device answers and goes on playing
whatever it was. Nothing is selected and nothing is written.

This is what `presets show` and `presets export` run.

The reply carries no name. That comes from the listing, which is why reading one
slot costs two calls.

## What the device gives back is not a `.hlx`

A preset arrives as a MessagePack blob whose contents are three concatenated
MessagePack values: the magic string `l6-helix`, a 48-byte table of byte offsets
into the blob, and the preset map itself.

The `.hlx` JSON in [preset-format.md](preset-format.md) is a host-side format.
They are different representations of the same preset, and converting between
them loses whatever neither side models.

### The offset table, read off three presets

Twelve offsets, each pointing at where something starts. The first is the preset
map itself and the last two are the end of the document; the nine between point
at the key byte of one section, in an order that is not the order the sections
are written in:

| entry | points at                                             |
| ----- | ----------------------------------------------------- |
| 0     | the preset map                                        |
| 1-9   | sections `0`, `1`, `3`, `4`, `2`, `5`, `6`, `7`, `10` |
| 10-11 | the end of the document                               |

Identical across every capture. A write recomputes them from where each section
actually landed.

**The offset table is the hazard in any write.** The device seeks with it rather
than walking the MessagePack, so a re-encode that changes any field's byte width
shifts every offset after it. MessagePack allows several encodings of the same
integer and the device emits wide tags where a naive encoder emits narrow ones.
tonepush measured 91 of 103 wide tags shrinking in one preset. The device
accepts such a write and then reads the preset as empty. Both projects lost
hardware sessions to this before fixing it.

Measured here rather than taken on trust. Decoding one of these presets and
encoding it again with an ordinary MessagePack encoder changes its length by
+99, +135 and -25 bytes on the three captures in `pkg/sdk/wire/testdata`. Every
offset after the first change would point at the wrong byte.

`wire.Document` is the answer: it keeps every section as the bytes the device
sent, writes the magic and the table at the widths the device used, and
recomputes the offsets from where the sections land. A preset read and written
back through it is the same bytes, which is asserted over all three captures.
Changing one section moves only what follows it.

The consequence for this project is larger than a bug: writing a `.hlx`
synthesised from nothing is the least solved problem in the whole space, and
neither project does it. The reliable shape is **read a preset off the device,
mutate it, write it back**, which is also what
[the RigSpec design record](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md)
concluded from a different direction, and what the corpus said when it showed
98.6% of real presets carrying routing that a generated one has none of.

## Inside the preset map

Everything is numbered. The keys below are what an HX Stomp on firmware 3.71
answered; nothing in Line 6's files documents them.

| key  | holds                                                           |
| ---- | --------------------------------------------------------------- |
| `0`  | the tone: `22` is the chain, as a fixed-length array of entries |
| `3`  | footswitches: `8` is a list of switches, in order               |
| `7`  | metadata: the firmware version the preset was written by        |
| `10` | snapshots: `10` is the list of them                             |

A chain entry is `{19: kind, 20: body}`. Kind `6` is a block somebody placed;
every other kind is the device's own: an input, an output, a gap where nothing
sits.

An entry that is not a block is the device's own, and each is one of four
things:

| kind | holds                                                    |
| ---- | -------------------------------------------------------- |
| `0`  | the input: `5` is which one, and three parameters follow |
| `1`  | the main output: `6` is which one, and two parameters    |
| `2`  | the second input under `14`, and the split under `15`    |
| `3`  | the second output under `16`, and the join under `17`    |

A split and a join name their own model under `8`, because more than one kind of
split exists. An input and an output name none: the device knows which are its
own, so the catalog carries them per device, read from the same io.models file
that lists which devices each belongs to.

**The position in that array is the block's number, and it is not its place in
the chain.** A device lays blocks on a fixed grid and leaves gaps: a preset
holding four blocks can have them at 5, 6, 8 and 13. Footswitch assignments
address blocks by that number, so renumbering them to 0 through 3 breaks the
only link between a switch and the block it works on. The body holds:

| key         | holds                                         |
| ----------- | --------------------------------------------- |
| `24` → `25` | the model, as a number, see below             |
| `11` → `4`  | the parameters, as a bare array with no names |
| `12` → `4`  | the cabinet an amplifier carries with it      |
| `10`        | whether the block is switched on              |

An amplifier and its cabinet are one block to a device and two entries in a
preset: the amplifier with a `@cab` pointing at a sibling, and the cabinet under
that name. 304 of 721 HX Stomp presets in the corpus have one.

Key `12` holds the cabinet's settings, and `3` beside them says how many the
cabinet model has names for. Anything past that is the microphone. Which cabinet
is not in the answer, because an amplifier names the one Line 6 voiced it with
and the catalog carries that.

A preset also records what kind of block each is, which a device leaves implied.
Measured over the corpus with no exceptions: an amplifier alone is 1, an
amplifier carrying a cabinet is 3, a cabinet is 2, everything else is 0.

A footswitch entry is `{10: ordinal, 11: body, 16: colour}` inside the list at
`3` → `8`. The list position is the switch, so the first group is FS1, and a
switch can carry more than one entry when it toggles several blocks.

`16` is the colour somebody chose, as a position in the device's own list, and
`0` is "Auto Color", where the light follows the block instead. Confirmed
against hardware: a switch set to Green in HX Edit reports `6` and one set to
Violet reports `9`.

The body's `5` is the label the pedal prints. Its `6` is *not* the switch colour
It is the block's own, the same for every block of that kind, which is why it
stays put when somebody changes a light. That is a trap worth naming: `6`
correlates with the colour so strongly on untouched presets that it reads as
correct until somebody sets one.

The names for those positions come from `HelixControls.json`, under
`footswitchLED`, and are generated into the catalog rather than written down. A
firmware that adds a colour would otherwise be reported under the wrong name.

A snapshot carries `4` as its name, `5` as its tempo and `12` as its colour.

## Model numbers are an index into HX Edit's own table

A block names its model with a number, and that number is a position in
`Helix.sym`, a plain JSON file in HX Edit's resources listing **833** symbols,
each with its parameters **in the order the device sends their values**.

That file is what makes a preset off the hardware readable: without it a block
is a number and its settings are an unlabelled array.

833 is larger than the 681 models in Line 6's `.models` files because the table
holds a mono and a stereo entry for the same model. Trimming that suffix joins
813 of them to a catalog block; the remaining 20 are hardware an HX Stomp does
not have, a second effects loop or the flow inputs of a bigger Helix, and keep
their own name so a rig still rebuilds them exactly.

The table is generated into the catalog, so it ships in the binary rather than
being read at run time. A catalog generated before this existed has none, and
`presets show` against hardware says so rather than guessing.

## Writing a preset

Opcode 5 writes a document into a slot and leaves its name alone. Opcode 8
writes one and names it, which is what a paste or an import does.

```text
opcode 5   {102: txn, 100: 5, 101: {107: setlist, 108: slot,
                                    123: false, 124: false, 125: 0,
                                    110: document}}
opcode 8   the same, with 109: name
```

Keys 123, 124 and 125 go out with every write and come back unchanged. Nobody
has established what they mean; every capture carries `false`, `false` and `0`.
A preset listing carries the same trio.

Three rules, and a device punishes each of them.

**The document must be byte-exact.** One that differs in length from what its
offset table claims leaves those offsets pointing at the wrong places. The
device accepts the write and then reads the preset as empty. `wire.Document` is
what keeps that from happening.

**A message goes out in pieces.** A device takes 256 bytes of stream data per
frame and paces the sender with acknowledgements. Sending a whole preset at once
fills its receive window and stalls the endpoint: the transfer times out, and
the interface will not be claimed again until the device is power cycled.

**A write is not finished when it is accepted.** The reply carries status 1,
meaning the device took it, and completion arrives later as a notification
carrying the same transaction with status 0. A client that treats the first as
the end races its next write against a commit still running. A device tolerates
about a dozen of those and then stops accepting writes at all.

None of this has been sent to hardware from here. It is tested against a
scripted device, which establishes that the message is built and paced correctly
and nothing about whether the device likes it.

## Opening a session

```text
claim interface 0 → release → claim again      # HX Edit does this; so must we
drain until three consecutive reads time out
control: hello, open service 5, ack, close, READ, hello, open service 2, ack
events:  hello, open service 4, ack
data:    hello, open service 6, ack
```

The claim-release-claim looks like startup noise. It is not: the device carries
channel state across connections, and the release is what clears it.

The control channel is opened **twice**, from scratch, and talks on the second.
Requests sent to service 5 time out silently, which is easy to mistake for a
flaky device.

### Two things found here, on hardware

Neither appears in the sources this was written from, and both cost a session to
find.

**The device must be read after the close, before the reopen.** Sending the
close and the new opening back to back leaves it still talking about the channel
that went away: it never answers the reopen, `ack` stays at `0x1000` where it
should be `0x1009`, and the first request is then ignored without any error. The
symptom is a handshake that reports success followed by a request that times
out.

**`SetAutoDetach` must not be called on macOS.** Detaching a kernel driver is a
Linux concept; macOS refuses it with `libusb: bad access`, which reads like a
permissions problem and is not one. Interface 0 is vendor-specific and has no
driver bound on any platform.

`TONESTACK_USB_DEBUG=1` traces every frame in and out, which is how both of
these were found.

## Listing presets

Opcode 1, on the control channel:

```text
request   {102: txn, 100: 1, 101: {107: setlist, 101: 2}}
reply     {102: txn, 103: 0, 104: [ {index: {109: name, …}}, … ]}
```

The `101: 2` inside the arguments is a selector whose meaning is not known. It
is sent because HX Edit sends it.

Argument order is **not sorted**, 107 precedes 101, and integers go out in the
narrowest unsigned form that holds them, so 1000 is a three-byte `uint16` and 2
is a single byte. Both are what HX Edit emits, and matching it keeps a call
byte-identical to one the device is known to accept.

The reply holds one entry per slot: a map of exactly one pair, from an index to
a detail map whose key 109 is the name. **Read entries by position, not by that
key.** The key is the index a preset had before it was last reordered on the
pedal, and no command accepts it as an address.

Names are C strings whose declared length counts a trailing NUL, so every one
arrives a byte longer than it reads.

An HX Stomp answers with 126 entries; a Helix and an HX Stomp XL answer 128.

## Rules that keep a device alive

Learned by other people the hard way. Ignoring any of them risks hardware that
needs its power supply physically pulled.

1. **Never call USB reset.** `libusb_reset_device` takes an HX Stomp off the bus
   and it does not come back without a physical unplug.
2. **Always have a read posted.** The device sends notifications unasked. With
   nothing draining the IN endpoint its outgoing queue fills, at which point it
   stops draining the incoming endpoint too and the next write times out.
3. **Pace deferred work on the completion notification.** Racing commits is
   tolerated about a dozen times and then writes stop being accepted.
4. **Handshake once per session.** Repeating it on an open channel wedges the
   device. A timeout is to be reported, not retried by reconnecting.
5. **Let flash settle.** A burst of renames once corrupted a setlist past what a
   power cycle could clear.
6. **Quit HX Edit first.** It holds interface 0 exclusively.

A wedged device needs the 9V adapter unplugged. USB alone is not enough, because
the unit stays powered and keeps its session across a replug.

## Model data stays on the machine that owns it

Model names, parameter ranges and artwork are Line 6's, shipped inside HX Edit
as `HelixModelDefs.bin` and `HX_ModelCatalog.json`. Both reference projects read
them from the user's own installation at run time and refuse to redistribute
them. This project generates its catalog the same way. See
[catalog.md](catalog.md).
