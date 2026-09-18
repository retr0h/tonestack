# Create a rig for a player

The whole path, from a name to a file that loads.

### 1. Find out whether the gear exists

Before writing anything. A rig naming an amplifier no device models is only
discovered at build time, and by then the name has usually been copied somewhere
else.

```bash
tonestack catalog list --search ampeg
tonestack catalog list --subcategory bass --category amp
```

Names are real-world gear. "Ampeg SVT", "Klon Centaur", never model identifiers.
[catalog.md](../catalog.md) explains where that mapping comes from and why it
exists.

### 2. Find out what else belongs in the chain

```bash
tonestack corpus show --instrument bass
```

This is measured over 4,324 real presets, not asserted. It says a bass chain
holds a compressor 88% of the time and that drive sits ahead of the amp in 88%
of the chains that have one. You do not have to act on it, since the build fills
in what is near-universal and says so, but it tells you what a complete chain
for that instrument looks like.

### 3. Write it

```bash
tonestack recipes new \
  --id mike-dirnt --name "Mike Dirnt" --band "Green Day" \
  --instrument bass --amp "Ampeg SVT" --cab "Ampeg 8x10"
```

Every gear name is resolved before anything is written, and near misses are
suggested when one does not resolve.

The file goes to your own recipes directory, `$XDG_DATA_HOME/tonestack/recipes`,
or `~/.local/share/tonestack/recipes` when that variable is unset, and the
command prints its path. `recipes list`, `recipes show`, `presets make` and the
MCP server read that directory beside the rigs that ship, so the new rig builds
straight away. `--dir` writes somewhere else instead. A rig written there is
only found when you name the same directory again, with `--dir` or with
`presets make --recipes`, and the rigs that ship are read beside it either way.

`--kind` changes what a copy made with `--from` is attributed to, and is refused
without `--from`. A rig written from gear is always an artist.

Then open the file. The flags write the chain; everything that makes the rig
worth reading is what you add next.

| field           | what it carries                                  | why it is not optional in practice                                                  |
| --------------- | ------------------------------------------------ | ----------------------------------------------------------------------------------- |
| `evidence`      | where each gear claim came from, per chain entry | a chain with no sources is a model's guess wearing a filename                       |
| `confidence`    | how much of the above you believe                | `low` is an honest answer and the tool displays it as unverified                    |
| `subject.era`   | when the rig applied, in words                   | a rig that does not say when it applies claims to be timeless and usually is not    |
| `subject.years` | the same period as numbers                       | nothing can be measured for this rig without it: records are held to these years    |
| `played`        | the instrument and its strings                   | a figure measured without it is attributed to the amplifier, which did not cause it |
| `technique`     | pick or fingers, and where on the string         | a picked record tuned by ear sounds dull played fingered                            |
| `character`     | how it should sound, each word with its evidence | this is what moves controls when the preset is built                                |

[recipes.md](../recipes.md) has a section per field and
[`examples/rigspec/mike-dirnt.yaml`](../../examples/rigspec/mike-dirnt.yaml)
shows all of them on one subject. Fill what you can source and leave the rest
out: an empty field is a gap somebody can close, and a filled one nobody checked
is a claim this project will repeat back as fact.

**Be honest about where the gear came from.** `kind: llm` on a piece of evidence
means a model asserted it and nobody checked. That is reliable for well-known
players, unreliable for obscure ones, and the model cannot tell which it is
doing, which makes it the largest correctness risk here.

**If this player will ever be measured**, the records go in
`resources/music/<instrument>/<id>/`, and `<id>` is this rig's identifier. That
directory name is the only thing joining a rig to its records: spell it
differently and the rig reads as one nobody has measured, while the records read
as belonging to nobody. `tonestack recipes records --corpus <dir>` says so when
it happens. [Add records to a corpus](add-records-to-a-corpus.md) is the rest.

### 4. Build it

```bash
tonestack presets make --id mike-dirnt --out mike.hlx
```

The output is the point. It reports every block chosen, what real gear each one
emulates, what it costs, how much of the processor is used, anything added that
the rig did not ask for, and what each word in the rig's `character` did:

```text
  ●  0.0  Deluxe Comp       Line 6 Original             1.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  ████████░░░░░░░░░░░░░░░░  35.6%

  added Deluxe Comp — almost every chain has one (88% of chains)

  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.72
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — Attack 0.04 to 0.05
```

[recipes.md](../recipes.md#character-describes-the-result-not-the-control)
explains the `heard` lines.

Read it before you plug anything in. A wrong amp is a bad miss that nothing
downstream recovers from, and it is visible right there.

### 5. Get it onto the device

See [below](get-it-onto-the-device.md).
