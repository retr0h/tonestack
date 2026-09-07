# Talking to a device

Two ways to reach an HX Stomp, and the reason only one of them carries presets
in both directions.

## Reading is live; writing is not

`presets list`, `show` and `export` talk to the device over USB. Nothing is
selected, loaded or written — the device answers and goes on playing whatever it
was:

```bash
tonestack presets list
tonestack presets show   --slot 31A
tonestack presets export --slot 31A --out lead.yaml
```

Putting a preset *onto* a device still goes through a file HX Edit wrote:

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
changes any field's byte width shifts every offset after it — and the device
accepts such a write and then reads the preset as empty. Both projects lost
hardware sessions to exactly this.

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

cgo is decided by the **import graph**, not the module boundary — nothing that
avoids importing `pkg/sdk` pays for it. A future HTTP service never touches a
device and stays pure Go.

## Prerequisites

```bash
brew install libusb        # macOS
apt-get install libusb-1.0-0-dev   # Debian/Ubuntu
```

`github.com/google/gousb` binds to it. On macOS no special privileges are
needed: enumeration reads descriptors only and never opens a device.

## Two identifier systems, deliberately separate

A device answers to a **USB product ID** on the bus and is named by a different
**device ID** inside a preset. They are unrelated numbers and both are needed.

| Device      | USB `vid:pid` |
| ----------- | ------------- |
| HX Stomp    | `0e41:4246`   |
| HX Stomp XL | `0e41:4253`   |
| Helix Floor | `0e41:4248`   |
| Helix LT    | `0e41:424a`   |

The other identifier — the integer a preset carries in `data.device` — is listed
in [preset-format.md](preset-format.md). `sdk.Models` holds both, because
resolving one to the other is exactly this package's job.

`sdk.Models` holds the mapping. Product IDs were observed on the bus; device IDs
come from the preset corpus. A device missing from that table is still reachable
over USB but will not be named, and presets cannot be written for it.

## Hardware stays behind an interface

Discovery is a pure function over a \[`Lister`\], so it is fully tested with a
fake and no device attached:

```go
type Lister interface {
	List(ctx context.Context) ([]Descriptor, error)
}
```

`pkg/sdk/usb.go` holds the only libusb-backed implementation and is listed in
`.coverignore`, because covering it would require hardware in CI. **Keep it
thin.** Anything with a decision in it belongs on the other side of the
interface where a test can reach it. If that file grows past enumeration and
transfer, the logic has leaked into the untestable half.

## Builds without cgo still work

`usb.go` carries `//go:build cgo`; `usb_nocgo.go` provides the same surface for
builds without it, returning `ErrNoUSBSupport` from every call.

That keeps `go install` working for someone who has no libusb. They get
everything except device access — describing a chain, validating it and writing
a preset are all pure Go — and a clear message rather than a link error if they
try to reach hardware.

CI installs libusb so `pkg/sdk` is compiled, vetted and linted like everything
else. Running CI with `CGO_ENABLED=0` would avoid the system dependency but
would leave that package unchecked anywhere except a developer's machine.
