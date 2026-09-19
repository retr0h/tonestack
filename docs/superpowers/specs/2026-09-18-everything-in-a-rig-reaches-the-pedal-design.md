# Everything in a rig reaches the pedal

2026-09-18

## The problem

A rig can say a thing that changes nothing, and nothing says so.

Building all nine rigs and reading what the compiler reports back: sixteen
character words are declared across them and eight of those move no control.
Four are deliberate and four are broken, and from the outside they are
identical. Both print the same shrug.

```
bootsy-collins   heard envelope-swept — nothing acts on this yet
bootsy-collins   heard mid-forward — the Ampeg B-15NF has no Mid
```

The first is by design. The vocabulary file says so at the top: decay,
string-noise, pickup and movement "describe the player and the instrument, and
move nothing". The second is a defect: `mid-forward` is the one claim in
Bootsy's rig earned by measuring records, and it dies because the amplifier
standing in for his Alembic has no Mid control and nothing reaches for an EQ.

Telling them apart requires reading the source. An agent filed the first as a
bug this week, which is the evidence that the distinction is invisible.

The same hole runs wider than the words. `technique` and `strings` are carried
by all nine rigs, displayed by the CLI, and read by nothing that builds a
preset. `requires` names third-party content a device may not have, and no code
reads it at all, so a rig depending on an impulse response nobody owns builds
without complaint and sounds wrong.

## The rule

A field earns its place in the contract one of two ways:

1. **It changes the preset.** It becomes a block, or it moves a control.
2. **It protects a claim that changes the preset.** Evidence decides whether a
   word is allowed to move a knob at all. The era check stops a figure derived
   from the wrong decade's records from moving one.

Anything in neither category comes out.

The rule as it was put: "RigSpec should only contain what we can actually
change on the pedal", with one carve-out argued for rather than assumed. Taken
literally the rule deletes the evidence layer, because a citation moves no knob.
It would also delete `subject.years` and `played[].records`, which exist to
catch a measured figure taken from a record made on other gear. Those are the
machinery that decides whether a knob may move, and deleting them leaves the
knobs moving on worse information. Hence the second category, and nothing else
gets in on that ticket.

## Where each field lands

| Field                                                                                              | Today                                    | Decision                                |
| -------------------------------------------------------------------------------------------------- | ---------------------------------------- | --------------------------------------- |
| `chain[].gear`, `settings`, `substitute`                                                           | becomes blocks and knobs                 | keep                                    |
| `character`, 16 of 24 terms                                                                        | moves controls                           | keep                                    |
| `blocks`, `controllers`, `footswitches`, `snapshots`, `sections`, `target`, `device`, `instrument` | reach the preset                         | keep                                    |
| `character`, the mids axis                                                                         | **broken**: dies when the amp has no Mid | fix, see below                          |
| `character`, the movement axis                                                                     | inert                                    | wire to block selection                 |
| `character`, decay / string-noise / pickup axes                                                    | inert by design                          | **delete**, six terms                   |
| `technique`                                                                                        | displayed only                           | wire to block selection                 |
| `played[].strings`                                                                                 | displayed only                           | **delete the field**, keep the research |
| `played[].gear`, `played[].records`                                                                | never reaches the preset                 | keep, category 2                        |
| `requires`                                                                                         | read by nothing                          | make it refuse                          |
| `evidence`, `confidence`, `caveat`, `mutations`                                                    | inert                                    | keep, category 2                        |

## The single cause

Three of the four defects are one missing capability. Nothing in a rig can cause
a block to exist.

`fill` picks blocks from corpus statistics alone. Its signature takes the
blocks, the catalog, the statistics and the instrument, and never the rig, so no
word and no technique can ask for anything. That one gap explains all of it:

- `mid-forward` dies because nobody adds an EQ when the amp has no Mid.
- `envelope-swept` is inert because it cannot require a filter.
- `technique: slap` is inert because it cannot require a compressor.
- `requires: ir` is inert because it cannot check anything.

So the change is not four fixes. It is one new idea with four consequences.

### The division of labour

The design that falls out is worth stating plainly, because it is the thing to
keep when the details change:

**A word that names a block asks for the block. A word that names a quality
moves its knobs.**

`technique` and the movement axis are about what is in the signal chain, so they
feed block selection. The other five axes are about how the chain is set, so
they feed control values, as they do now. Nothing does both, which is what stops
a term being counted twice.

## What changes

### 1. A rig can require a block

`fill` gains the spec. Before the corpus statistics are consulted, the rig's own
demands are collected:

- `technique: slap` requires a compressor. Slap without one is not the sound,
  and the corpus agrees anyway on bass, so this mostly formalises what already
  happens and makes it stop being luck.
- `envelope-swept` requires a filter.
- A term whose control exists on no block in the chain requires a block that has
  it. This is what rescues `mid-forward`: with no Mid anywhere, an EQ is added
  and the word lands on its Mid.

A requirement the device cannot satisfy is reported against the rig, not
silently dropped.

### 2. `requires` refuses

A rig naming an impulse response or purchased model gets checked at build. If
the catalog cannot satisfy it the build refuses and says which one, rather than
producing a preset that will be silent or wrong in that block.

No rig uses this field today, so this costs nothing now and closes the hole
before the first one does.

### 3. Six terms are deleted

The decay, string-noise and pickup axes come out of `character-terms.json` and
out of the contract's vocabulary.

Only three of the six are used, by two rigs: `short-decay` and `quiet-strings`
on Pino Palladino, `bridge-forward` on Jaco Pastorius. The other three
(`long-decay`, `audible-strings`, `neck-forward`) are used by nobody.

Deleting a term deletes sourced work, so the evidence moves rather than dying.
Where a deleted term carried a citation, that citation attaches to the character
term it actually supports, as a `caveat`. Pino's `short-decay` came from the
album's engineer; that engineer's words explain why the record reads the way it
does, and belong beside the word that does move a knob.

The loss is real and worth naming: the rig can no longer record that Jaco played
near the bridge. That is true, it is interesting, and no pedal has a control for
where a hand is. It belongs in prose about the player, not in a format whose job
is to produce a preset.

### 4. `strings` is deleted, the research is not

`strings: round` and `strings: flat` come out of `played`.

This is the uncomfortable one. A week went into establishing Bootsy's Rotosounds
and correcting Flea's rounds to flats, and the field is the visible result. But
string winding changes the preset through nothing: the device models the
amplifier, not the instrument. What flats actually explain is why a player reads
dark, and `dark` already moves a control on measured evidence.

So each rig's strings citation moves onto the highs-axis term it explains. A rig
with no such term keeps the citation on the rig's own evidence list. The sources
survive, attached to the claim they support rather than to a field nobody reads.

**Flagged for review.** This is the one decision in this document worth
overruling. Keeping `strings` as documentation is defensible, and the rule above
is what argues it out. If the research feels worth its own field, say so and it
stays, with the contract saying plainly that it is documentation.

### 5. The message stops saying "yet"

"nothing acts on this yet" describes unfinished work. After this change there is
no such category: a word either moves something or does not exist. The remaining
reasons are real and each says what happened, for example that the device has no
block carrying the control a word needs.

### 6. The test that makes this stick

The point of the whole exercise:

**Every rig is built, and the build fails if any declared thing moved nothing.**

Concretely, a test that compiles all nine rigs and asserts that no character
term comes back without a parameter, and that every term in the vocabulary is
reachable. A term added to `character-terms.json` with no control behind it
fails the gate. A field added to the contract that nothing reads fails review
because this test is where somebody looks.

This is what stops the class of bug rather than the four instances of it. It is
also why the deletions matter more than they look: with no third state, "this is
deliberately inert" is not available as an excuse, and the next person cannot
add one.

## What this does not do

It does not make the preset correct. It makes the rig honest about what it asked
for. Whether an added EQ's Mid at the position `mid-forward` computes actually
sounds like Bootsy is unanswerable here and stays unanswerable until something
can hear, which is its own task and deliberately last.

It does not touch the evidence layer, the era check or the corpus.

## Migration

Nine rigs. Three lose a character term, all nine lose `strings`, none uses
`requires`. Bootsy, Jaco, McCartney and Pino gain an EQ block where a mids word
had nowhere to land, which changes their built presets: that is the fix working,
and each is worth listening to before it lands.

`docs/rigspec.md` and `docs/catalog.md` are generated and will follow.
`docs/recipes.md` and the RigSpec contract's own prose are hand-written and need
editing.
