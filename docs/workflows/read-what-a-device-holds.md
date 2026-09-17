# Read what a device holds

```bash
tonestack presets list
```

That reads the attached device over USB. **Quit HX Edit first.** It claims the
editor interface exclusively and nothing else can talk to the device while it
runs.

It is read-only: it asks the device to describe a setlist and nothing more.
Nothing is selected, loaded or written. [protocol.md](../protocol.md) covers
how, and carries the rules that keep a device alive if you are working on that
code.

`presets show` and `presets export` read the hardware too, and give the slot
back as a rig:

```bash
tonestack presets show   --slot 31A
tonestack presets export --slot 31A --out lead.yaml
```

A slot is addressed the way the pedal labels it, `01A` through `42C`. A bare
number works too, for scripts.

Every reading command also takes `--file`, for a backup HX Edit wrote when no
device is attached:

```bash
tonestack presets list --file device.hlb
tonestack presets show --file device.hlb --slot 31A
```

Putting a preset *onto* a device is the part that is not live. See
[Get it onto the device](get-it-onto-the-device.md).

A rig read off the device carries its routing, so compiling one puts the
device's own inputs, outputs, split and join back. Controller assignments are
the exception: nothing has decoded them yet, and a rig read over USB carries
none. Reading the same slot out of a backup carries them.
[protocol.md](../protocol.md) states what the device answers today and what is
still unknown.
