# Get it onto the device

Three ways, depending on what you have open.

**Straight to the device.** Plug in the Helix and quit HX Edit:

```bash
tonestack presets import --preset mike.hlx --slot 07A
```

```console
  kept ~/.local/state/tonestack/presets/07A-s0-20260910-041500.129384756.hlx

  Mike Dirnt → 07A

  written
```

A device has no undo, so whatever the slot held is read and saved first, and the
output says where. Put it back with `presets import --preset` and that file.
`--backup-dir` changes where they go. A backup never replaces a file already in
that directory. One that can't be written in full, or whose directory can't be
synced to disk, leaves no file behind, and nothing is written to the pedal.

Ctrl-C does not cut a write off halfway, because a half-sent message stalls the
pedal. The write finishes, then the command tells the pedal the session is over
and waits for it, which can take up to about 20 seconds. The first Ctrl-C prints
that the pedal is being let go safely, and the second prints that it is still
finishing. A third quits on the spot. That leaves the pedal believing an editor
is still attached, and it may need its power unplugged to recover, as
[protocol.md](../protocol.md#rules-that-keep-a-device-alive) explains. Every
command that talks to the device answers Ctrl-C this way, not only `import`.

`tonestack mcp start` answers Ctrl-C the same way while it holds the pedal. It
holds the pedal from an agent's device tool call until the agent has made none
for 10 seconds. A Ctrl-C at any other time prints nothing, and the server stops
straight away.

A slot with no blocks is kept as a `.bin` file instead. It holds the bytes the
device sent, so nothing is lost. `presets import --preset <file>.bin` puts one
back: the bytes go to the slot exactly as they came off, and the slot keeps the
name it has, because a `.bin` carries none.

**Through HX Edit.** `HX Edit → Import` and choose the `.hlx`.

**Into a backup**, with no device attached:

```bash
tonestack presets import --file device.hlb --preset mike.hlx \
  --slot 07A --out edited.hlb
# HX Edit → Restore
```

Nobody has yet confirmed that a device loads a generated preset and it sounds
right. The build validating against the catalog is the only claim this project
can make today; [device.md](../device.md) covers what is known about writing.
