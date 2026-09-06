# Writing a recipe

A recipe is curated knowledge about how a sound is built — which gear a player
uses, and how it should behave. It is the input to generation; a
[RigSpec](../schemas/README.md) is the output.

Recipes are the only data in this repository that is **ours**. The device
catalog and the gear map are derived from Line 6's own files and cannot be
shipped; these are hand-written and can.

## Name gear, never model identifiers

```yaml
rig:
  amp: Ampeg SVT        # not HD2_AmpSVBeastNrm
```

[`schemas/gear-map.json`](catalog.md) resolves the name to whichever model a
given device carries. Writing the identifier directly would tie the recipe to
one device, break when Line 6 renames a model, and make the file unreadable to
the person most able to correct it.

## A complete example

`recipes/artists/mike-dirnt.yaml`:

```yaml
id: mike-dirnt
kind: artist
name: Mike Dirnt
band: Green Day
instrument_type: bass

rig:
  instrument: Fender Precision Bass
  amp: Ampeg SVT
  cab: Ampeg 8x10
  pedals: []
  technique: pick, near the bridge

character:
  - mid-forward, not scooped
  - minimal drive, grit only on hard attack
  - tight low end, short decay

variants:
  - id: longview
    name: Longview
    character:
      - fingers rather than pick
      - rounder, more low-mid

provenance:
  source: llm
  confidence: medium
  notes: Gear identification is unverified against any rig rundown or interview.
```

The contract is [`schemas/recipe.schema.json`](../schemas/recipe.schema.json)
Loading a recipe validates it: `pkg/recipe` refuses anything the contract
rejects, and a conformance test pins the shipped recipes to the published
schema.

## `rig` versus `variants`

The split is load-bearing. A player's gear is career-long; settings change per
song.

|                                           | Belongs in                  |
| ----------------------------------------- | --------------------------- |
| Instrument, amp, cab, pedals, technique   | `rig`                       |
| Drive amount, EQ curve, effects on or off | `character`, or a `variant` |

A wrong amp is a bad miss — nothing downstream recovers from it. A wrong drive
level is a near miss that one correction fixes. Recording them separately means
*"make it more like Longview"* moves only what should move.

## Writing `character`

Describe the **result**, not the control:

```yaml
character:
  - mid-forward, not scooped        # good — a claim about the sound
  - grit only on hard attack        # good — describes behaviour
  - set Mid to 0.7                  # bad  — that is the generator's job
```

The generator turns these into moves against the catalog's real parameter
ranges. Naming a control hard-codes a number that may not suit the model that
gets chosen.

## Provenance is not decoration

```yaml
provenance:
  source: llm          # llm | curated | cited
  confidence: medium   # low | medium | high
```

`source: llm` means a language model asserted this and nobody checked. That is
reliable for well-known players and unreliable for obscure ones, **and the model
cannot always tell which it is doing** — see [knowledge.md](knowledge.md).

Anything a person confirms becomes `curated`, which outranks `llm` permanently.
A recipe corrected against a real amp is worth more than any amount of generated
guessing.

## Correcting one

If it sounds wrong, change the line and the fix is permanent. That loop — hear
it, correct one field, never hear it again — is the product. Prefer correcting a
recipe over adjusting a generated preset, because only the recipe survives.
