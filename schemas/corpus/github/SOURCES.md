# Helix preset corpus — GitHub sources

Collected 2778 preset/setlist files from 32 public GitHub repositories. Files
were shallow-cloned, filtered to JSON documents whose top-level `schema` starts
with `L6`, deduplicated by SHA-256 across all repos, and copied here preserving
each repo's original directory layout under `<owner>__<repo>/`.

The 35 `.hls` / 2 `.pgs` setlist files and 1 `.hlb` bundle are zlib+Base64
containers; decoding them yields a further 3792 embedded presets (excluding
empty slots).

Everything here is redistributed from public repositories for format
reverse-engineering. Repos marked "no license file" carry no explicit grant —
attribution below is the record of origin.

| Repository                         | URL                                                 | License         | Files | + presets inside setlists | Devices                                                                                                                                                                                   |
| ---------------------------------- | --------------------------------------------------- | --------------- | ----- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `engageintellect/helix-tones`      | https://github.com/engageintellect/helix-tones      | no license file | 947   |                           | 2162694 (HX Stomp)=533, 2162689 (Helix Floor / Rack)=232, 2162944 (Helix Native)=142, 2162693 (HX Effects)=21, 2162699 (HX Stomp XL)=8, 2162692 (Helix LT)=7, 2424834 (Line 6 Catalyst)=4 |
| `novff/PersonalRepo`               | https://github.com/novff/PersonalRepo               | no license file | 721   | 1339                      | 2162689 (Helix Floor / Rack)=559, 2162692 (Helix LT)=53, 2162944 (Helix Native)=52, 2162690 (Helix (2nd hardware variant, likely Rack))=43, 2162694 (HX Stomp)=3                          |
| `b-vesco/vesco-helixpresets`       | https://github.com/b-vesco/vesco-helixpresets       | no license file | 438   |                           | 2162689 (Helix Floor / Rack)=438                                                                                                                                                          |
| `f3sty/Yamaha_THRII_presets`       | https://github.com/f3sty/Yamaha_THRII_presets       | no license file | 235   |                           | 2359298 (Yamaha THR-II)=235                                                                                                                                                               |
| `fargraph/pod-go-presets`          | https://github.com/fargraph/pod-go-presets          | MIT             | 124   |                           | 2162696 (POD Go (variant B))=124                                                                                                                                                          |
| `JesseWebDotCom/bobs-fault`        | https://github.com/JesseWebDotCom/bobs-fault        | no license file | 40    | 128                       | 2162689 (Helix Floor / Rack)=39                                                                                                                                                           |
| `tylerdrakemusic/Music`            | https://github.com/tylerdrakemusic/Music            | MIT             | 33    |                           | 2162694 (HX Stomp)=33                                                                                                                                                                     |
| `ruizjme/hx-effects`               | https://github.com/ruizjme/hx-effects               | no license file | 30    | 658                       | 2162693 (HX Effects)=21                                                                                                                                                                   |
| `dry-socket/HXman-preset`          | https://github.com/dry-socket/HXman-preset          | no license file | 28    | 896                       | 2162944 (Helix Native)=9, 2162694 (HX Stomp)=9, 2162692 (Helix LT)=2, 2162689 (Helix Floor / Rack)=1                                                                                      |
| `mthines/kaiju-helix`              | https://github.com/mthines/kaiju-helix              | no license file | 28    | 256                       | 2162689 (Helix Floor / Rack)=24, 2162944 (Helix Native)=2                                                                                                                                 |
| `Arenteria518/pod-go-tools`        | https://github.com/Arenteria518/pod-go-tools        | MIT             | 27    | 25                        | 2162695 (POD Go (variant A))=26                                                                                                                                                           |
| `Zavsek/POD_GO-Helix_converter`    | https://github.com/Zavsek/POD_GO-Helix_converter    | no license file | 24    |                           | 2162944 (Helix Native)=19, 2162695 (POD Go (variant A))=5                                                                                                                                 |
| `planbnet/hxgen`                   | https://github.com/planbnet/hxgen                   | no license file | 21    |                           | 2162694 (HX Stomp)=21                                                                                                                                                                     |
| `rdalin82/hx-stomp-patch-creator`  | https://github.com/rdalin82/hx-stomp-patch-creator  | no license file | 21    |                           | 2162694 (HX Stomp)=20, 2162692 (Helix LT)=1                                                                                                                                               |
| `thatmarkb/tmbsuperpatch`          | https://github.com/thatmarkb/tmbsuperpatch          | no license file | 14    |                           | 2162689 (Helix Floor / Rack)=1                                                                                                                                                            |
| `SeannyQuest/tonemaker`            | https://github.com/SeannyQuest/tonemaker            | MIT             | 11    |                           | 2162695 (POD Go (variant A))=11                                                                                                                                                           |
| `basstones/stomp_xl_presets`       | https://github.com/basstones/stomp_xl_presets       | no license file | 4     |                           | 2162699 (HX Stomp XL)=4                                                                                                                                                                   |
| `mthines/hx-ctrl`                  | https://github.com/mthines/hx-ctrl                  | no license file | 4     |                           | 2162689 (Helix Floor / Rack)=2, 2162944 (Helix Native)=2                                                                                                                                  |
| `noseglasses/MatchPatch`           | https://github.com/noseglasses/MatchPatch           | MIT             | 4     | 49                        | 2162696 (POD Go (variant B))=3                                                                                                                                                            |
| `GetPsycho/HX-IA-generator`        | https://github.com/GetPsycho/HX-IA-generator        | no license file | 3     | 92                        | 2162693 (HX Effects)=2                                                                                                                                                                    |
| `HackLabsGuitar/helix-py-api`      | https://github.com/HackLabsGuitar/helix-py-api      | BSD-3-Clause    | 3     | 128                       | 2162689 (Helix Floor / Rack)=1                                                                                                                                                            |
| `crmne/tonepush`                   | https://github.com/crmne/tonepush                   | MIT             | 3     | 116                       | 2162694 (HX Stomp)=1                                                                                                                                                                      |
| `lmeadors/helix-catalog`           | https://github.com/lmeadors/helix-catalog           | no license file | 3     |                           | 2162689 (Helix Floor / Rack)=2, 2162692 (Helix LT)=1                                                                                                                                      |
| `agarat/openpodgo`                 | https://github.com/agarat/openpodgo                 | MIT             | 2     |                           | 2162695 (POD Go (variant A))=2                                                                                                                                                            |
| `johnsherlock/HelixSetlistEditor`  | https://github.com/johnsherlock/HelixSetlistEditor  | no license file | 2     | 105                       | —                                                                                                                                                                                         |
| `sheax0r/helixgen-core`            | https://github.com/sheax0r/helixgen-core            | MIT             | 2     |                           | 2162689 (Helix Floor / Rack)=2                                                                                                                                                            |
| `MrCitron/helaix`                  | https://github.com/MrCitron/helaix                  | MIT             | 1     |                           | 2162689 (Helix Floor / Rack)=1                                                                                                                                                            |
| `bb-joelle/helix-preset-generator` | https://github.com/bb-joelle/helix-preset-generator | no license file | 1     |                           | 2162944 (Helix Native)=1                                                                                                                                                                  |
| `guyburton/HelixAudioClipLooper`   | https://github.com/guyburton/HelixAudioClipLooper   | Apache-2.0      | 1     |                           | 2162692 (Helix LT)=1                                                                                                                                                                      |
| `rachokthebot-dev/music-apps`      | https://github.com/rachokthebot-dev/music-apps      | MIT             | 1     |                           | 2162692 (Helix LT)=1                                                                                                                                                                      |
| `sgreer81/hx-stomp-ai-skills`      | https://github.com/sgreer81/hx-stomp-ai-skills      | MIT             | 1     |                           | 2162694 (HX Stomp)=1                                                                                                                                                                      |
| `turtelduo/helix`                  | https://github.com/turtelduo/helix                  | no license file | 1     |                           | 2162689 (Helix Floor / Rack)=1                                                                                                                                                            |

## File-type breakdown

| Extension | Count | What it is                                                                            |
| --------- | ----- | ------------------------------------------------------------------------------------- |
| `.hlx`    | 2314  | Helix / HX single preset                                                              |
| `.thrl6p` | 235   | Yamaha THR-II preset                                                                  |
| `.pgp`    | 167   | POD Go preset                                                                         |
| `.hls`    | 35    | Helix setlist (compressed bundle of 128 preset slots)                                 |
| `.fav`    | 14    | Model-favorites list (`L6ModelFavorite`) — enumerates model IDs, kept for the catalog |
| `.json`   | 6     | Preset stored with a .json extension                                                  |
| `.catl6p` | 4     | Line 6 Catalyst preset                                                                |
| `.pgs`    | 2     | POD Go setlist (compressed)                                                           |
| `.hlb`    | 1     | Helix bundle (compressed)                                                             |

## Schema breakdown

- `L6Preset`: 2726
- `L6Setlist`: 37
- `L6ModelFavorite`: 14
- `L6PresetBundle`: 1

## Device IDs observed

Names are inferred from the `data.meta.application` string, the file extension,
and the source repo (e.g. `basstones/stomp_xl_presets` pins 2162699 to HX Stomp
XL). `2162689` vs `2162690` both report application `Helix` and have not been
separated with certainty.

| Device ID | Inferred device                           | Standalone files | Inside setlists |
| --------- | ----------------------------------------- | ---------------- | --------------- |
| 2162944   | Helix Native                              | 227              | 2304            |
| 2162689   | Helix Floor / Rack                        | 1303             | 548             |
| 2162693   | HX Effects                                | 44               | 750             |
| 2162694   | HX Stomp                                  | 621              | 116             |
| 2359298   | Yamaha THR-II                             | 235              | 0               |
| 2162696   | POD Go (variant B)                        | 127              | 49              |
| 2162695   | POD Go (variant A)                        | 44               | 25              |
| 2162692   | Helix LT                                  | 66               | 0               |
| 2162690   | Helix (2nd hardware variant, likely Rack) | 43               | 0               |
| 2162699   | HX Stomp XL                               | 12               | 0               |
| 2424834   | Line 6 Catalyst                           | 4                | 0               |

## Not collected

Already gathered elsewhere and deliberately skipped:
`EmmanuelBeziat/helix-presets`, `mattbreit/hxstomp`, `niksoper/helix-presets`,
`bellol/helix-presets`, `cpwilkerson/helix-patches`,
`EgorNikitin1/Line6-HLX-Parser`, `sensorium/phelix`, `jdjfisher/helix-preset`,
`PhaseDog/HelixNativePresets`.

`john-baxter-dev/fretwire` (Rust HX editor) and `dbagchee/helix-preset-viewer`
contain no preset files but do embed model-ID tables in source — worth mining
separately.
