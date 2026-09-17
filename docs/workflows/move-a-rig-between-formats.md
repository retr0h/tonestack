# Move a rig between formats

RigSpec is what this project speaks, so it is what `export` writes. Reading a
slot still means reading a backup rather than the device:

```bash
tonestack presets export --file device.hlb --slot 3 --out slot3.yaml
tonestack presets compile --rig slot3.yaml --out slot3.hlx
```

Out and back. A preset read into a rig and compiled again is the preset it came
from. That is asserted over every HX Stomp preset in the corpus, so it is a
measurement rather than a claim.

Two things make that work, and both matter if you hand-edit a rig in between.

`slot3.yaml` is a rig, the same format `recipes new` writes and `presets make`
reads. A lifted rig records `models: { HX Stomp: HD2_... }`, the exact model
each piece of gear resolved to. **665 models share only 469 names**, and "Ampeg
SVT" matches both channels, so a rig carrying the name alone would rebuild into
a different preset. Delete that line and compiling falls back to resolving the
name, which is right for a rig you wrote and wrong for one you lifted.

Compiling writes the chain into an untouched preset the device itself wrote, so
the result carries the inputs, outputs, split and join a device expects. 98.6%
of real presets have them and one assembled from nothing has none. `--template`
uses a particular preset as that base instead.

For a faithful copy rather than a reading, `--as hlx` writes the device's own
file, which also carries the routing and snapshots a rig models but nobody
chooses.
