# testdata

Three slots as an HX Stomp sent them, and the ground truth for everything this
package claims about the format.

| File           | Slot  | What it is                                                                                               |
| -------------- | ----- | -------------------------------------------------------------------------------------------------------- |
| `preset.bin`   | `27B` | `BAS:SVT Nrm`. Six blocks, a standalone cabinet, and the one captured controller assignment              |
| `switches.bin` | `09A` | `B15 Eras`. Two amps carrying cabinets, and footswitches with names and colours somebody chose           |
| `empty.bin`    | `02B` | An unused slot. Also embedded as `data/hx-stomp.blank.bin`, which every generated preset is written into |

Firmware `v2.92-457-ge30d971`, which the presets carry in section 7.

`BAS_SVT Nrm.hlx`, the same slot exported from HX Edit, is what settled the
position mapping and the controller assignment. It is not committed: it is
somebody's own preset rather than ours.
