# Switch and rearrange

These need the Helix plugged in and HX Edit quit, or `--file` to work on a
backup instead.

```bash
tonestack presets select --slot 27B              # load it, like a footswitch
tonestack presets copy   --from 01A --to 02A     # 02A becomes a copy of 01A
tonestack presets swap   --from 01A --to 02A     # exchange the two
```

`select` writes nothing. `copy` and `swap` keep what the destination held first,
the same way `import` does. Moving a preset is a swap: when one of the two slots
holds no preset, the preset lands there and the slot it came from is emptied,
which is a move and invents nothing.

A swap of two slots that both hold no preset is refused, and nothing is kept or
written, because there is nothing to move. `copy` fills an empty slot and leaves
the source as it is.
