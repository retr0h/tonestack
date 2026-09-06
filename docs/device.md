# Talking to a device

How `pkg/sdk` reaches the hardware over USB.

`pkg/sdk` reaches the hardware over USB. It is the only package needing cgo,
which is why it is the only one that cannot be cross-compiled or built with
`CGO_ENABLED=0`.

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
