# helix-sdk — Design

> **Superseded in part, 2026-09-06.** This proposes `helix-sdk` as a separate Go
> module to keep cgo out of everything else. That reasoning was wrong: cgo is
> determined by the import graph, not the module boundary, so a `pkg/sdk`
> package in the single `tonestack` module costs non-importers nothing. The USB
> protocol findings, the staging, and the API sketch all still stand — only the
> module argument is retracted.

**Date:** 2026-09-06 **Status:** draft, not approved **Scope:** talk to an HX
Stomp over USB from Go, without HX Edit

## Why

Every verification step in this project currently needs a human: generate a
preset, hand it over, wait for someone to import it in HX Edit and load it on
hardware. That gap is why "the JSON validates" and "the device accepted it" have
stayed different claims all through phase 1.

An SDK closes the loop. Generate a preset, push it, read it back, compare —
unattended.

## Feasibility

Established by reading two working implementations:

| Project              | Language      | Licence | What it establishes                                    |
| -------------------- | ------------- | ------- | ------------------------------------------------------ |
| `kempline/helix_usb` | Python        | **MIT** | The full handshake, byte for byte. Portable.           |
| `Scrouik/hxlinux`    | Rust (`rusb`) | none    | That a complete editor is possible over this transport |

`hxlinux` ships no licence, so it is read for protocol facts only and no code or
data is taken from it. `helix_usb` is MIT and may be ported with attribution.

### The device

| Model       | VID:PID         |
| ----------- | --------------- |
| HX Stomp    | `0x0e41:0x4246` |
| HX Stomp XL | `0x0e41:0x4253` |
| Helix Floor | `0x0e41:0x4248` |
| Helix LT    | `0x0e41:0x424a` |

`0x0e41` is Line 6's vendor ID.

### The transport

```
bulk  OUT 0x01  /  bulk IN 0x81     control — commands and responses
bulk  OUT 0x02  /  bulk IN 0x82     secondary channel
iso   OUT 0x03  /  iso  IN 0x83     audio, not used here
```

Several interfaces must be claimed, including a MIDI interface. Connection is a
state machine: send a fixed packet, match the response prefix, send the next.
Three logical channels (`0x01`, `0x02`, `0x80`) each need a periodic keep-alive
or the device drops the session. Payloads are MessagePack.

Packets begin with a length, a type, and source/destination channel bytes;
sequence counters increment per channel.

## Shape

A separate Go module, because it is the only component needing cgo:

```
helix-sdk/
  pkg/device/     enumerate, open, claim, close
  pkg/transport/  bulk read/write, framing, sequence counters
  pkg/session/    handshake state machine, keep-alives
  pkg/protocol/   message encode/decode (MessagePack)
  pkg/helix/      the API consumers use
```

`gousb` (cgo, libusb) is the transport. **This is the one real cost.**
`helix-core`, `helixctl` and `helix-api` stay pure Go and statically linked;
only what imports `helix-sdk` inherits cgo. The web service never talks to a
device, so it never pays.

## The API

Small and honest about what it does. Reading is safe; writing is not, and the
API should make that visible at the call site.

```go
d, err := helix.Open(ctx)            // finds the first Line 6 device
defer d.Close()

info, err := d.Identify(ctx)         // model, firmware
names, err := d.PresetNames(ctx)     // every slot's name
p, err := d.ReadPreset(ctx, slot)    // raw .hlx bytes
err = d.WritePreset(ctx, slot, b)    // DESTRUCTIVE — see below
err = d.SetScribble(ctx, slot, text, colour)
```

`ReadPreset` returns the file verbatim so it can go straight into the corpus and
through the same parser as any downloaded preset. One code path.

## Staging

Riskiest last. Each stage is independently useful.

| Stage | Capability                           | Risk                    |
| ----- | ------------------------------------ | ----------------------- |
| A     | Enumerate, identify, report firmware | none — descriptor reads |
| B     | Read preset names, read a preset     | none — read-only        |
| C     | Write a preset to a slot             | overwrites user data    |
| D     | Scribble strips, colours, LEDs       | cosmetic                |

Stage B alone lets the corpus grow from the device itself and lets a generated
preset be compared against what the device stores. Stage C closes the loop.

**Before any write lands, the tool backs up the target slot.** A preset is user
data and losing someone's tone to a bug in our sequence counters is not
recoverable by apology.

## Testing

The device cannot be assumed present, so:

- **Transport and protocol** are tested against recorded traffic — capture real
  exchanges once, replay them in tests. No hardware in CI.
- **A `--device` build tag** gates the tests that need real hardware. They run
  on a developer machine with a Stomp attached, never in CI.
- **Round trip is the acceptance test:** read a preset off the device, write it
  back to a scratch slot, read it again, compare. If that holds, the transport
  and encoding are right.

## Later: MCP

An MCP server over this SDK is the point of the exercise. `list_presets`,
`read_preset`, `apply_preset`, `identify` as tools means an agent can verify its
own output against hardware instead of asking a person to. That is a transport
over the same library — the same shape as the CLI and the HTTP service, and no
change below it.

## Open questions

1. Does the Stomp expose the same endpoints as the Helix Floor that `helix_usb`
   was written against? The handshake may differ per model.
2. What does MessagePack carry — the whole `.hlx` document, or a binary form
   that must be converted?
3. Does writing a preset require a firmware-version match?
4. Is there a documented way to trigger the device's own backup, so a full
   corpus can be pulled in one operation rather than slot by slot?

None are answerable without a device attached.

## Not in scope

Editing individual parameters live, audio streaming, firmware update. This reads
presets, writes presets, and sets cosmetics. An editor is `hxlinux`'s job and it
already exists.
