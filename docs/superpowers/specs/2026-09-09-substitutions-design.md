# Substituting what a device cannot do

**Status:** implemented\
**Relates to:**
[2026-09-09-a-grammar-for-every-field-design.md](2026-09-09-a-grammar-for-every-field-design.md)

## The problem

A rig has to name gear the catalog resolves or it does not build. That is fine
until somebody's actual amplifier is not modelled, and then the rig has two bad
options: name something else and lie about what was played, or not exist.

Geddy Lee is the case that found this. Four rigs across four eras, and exactly
one of them is buildable:

| era       | played through       | catalog                       |
| --------- | -------------------- | ----------------------------- |
| 1974-81   | Ampeg SVT stacks     | modelled                      |
| 1982-87   | Wal, mostly direct   | nothing                       |
| 1993-2002 | Trace Elliot         | nothing                       |
| 2002-     | Orange AD200B stacks | Orange, but guitar heads only |

The same wall is at McCartney through a Vox AC100, and at anybody whose sound
came out of a preamp Line 6 never licensed.

Meanwhile the resolver already substitutes, in one hard-coded case: a cabinet
nothing emulates falls back to the one the amplifier was voiced with. It works,
it reports itself while building, and it is written down nowhere afterwards. So
the idea was already in the code, at the size of one special case.

## The shape

The rig goes on naming the real gear. What this device should put there instead
sits beside it.

```yaml
chain:
  - role: amp
    gear: Orange AD200B
    evidence:
      - kind: cited
        url: https://example.com/rig-rundown
    substitute:
      gear: Sunn Coliseum 300
      evidence:
        - kind: user
          url: https://reddit.com/r/Helix/...
          note: all-tube power section, closest thing the device carries
      confidence: low
```

The other way round was considered: name the buildable thing in `gear` and note
the real one beside it. It is easier to implement, because resolution never
needs a second path. It loses on everything else. The rig would assert something
false about the player, `gear` would mean one thing in some entries and another
in others, and the day Line 6 model the AD200B the fix is to rewrite `gear`,
move the citation and delete the block, rather than delete the block.

A rig outlives any one device. That is the whole argument for RigSpec over a
`.hlx`, and it settles this too.

## A substitution is a claim

Somebody saying two amplifiers are close enough is an assertion, and after
[the evidence rule](2026-09-09-a-grammar-for-every-field-design.md) every
assertion carries why it is believed. A forum post is `user`, an A/B somebody
recorded is `video`, a model guessing is `llm`, and the reader can tell which
they are looking at.

The one kind that was missing is where to *get* something. A stand-in can be
content that did not ship with the device, and a rig that names an impulse
response without saying where it comes from has named something the reader
cannot obtain. So `EvidenceKind` gains `store`, which its own description
invites: "a new way of learning is a new value here, not a new document". It is
the one kind answering "where do I get this" rather than "why is this believed",
and the description says so.

`requires` is unchanged and still does its job, which is different: it declares
what the *author* had loaded, in which slot, so somebody reading the rig knows
their slot 82 holds something else.

## Where it runs

Both directions, because a rig reaches a preset two ways and they have to agree:

- `Resolve`, building a chain from a rig, reports the substitution the way it
  already reports a filled-in block.
- `Lower`, writing a rig into a preset, takes the same path silently, because it
  has no channel to report on.

A stand-in nothing models is an error naming both:
`"Also Not A Thing" stands in for "Orange AD200B", and nothing emulates it either`.
Two names, because being told only that the substitute failed leaves the reader
wondering what it was standing in for.

## What this does not do

**It does not build a stand-in that is not in the catalog.** A purchased model
or a third-party impulse response cannot be resolved, because the catalog only
knows what shipped. Naming one as a substitute fails, and the error says so.
Making it work means writing an impulse response block against a slot, which
`catalog.NeedsUserIR` already knows how to recognise and nothing yet knows how
to write.

**It does not re-express the cabinet fallback as a substitution.** That one is
inferred rather than authored, and folding them together would mean either the
resolver writing into the rig or the rig carrying something nobody typed.

**It does not choose a substitute for you.** Nothing suggests that a Sunn is
close to an Orange. A person decides that, and the rig records who.

## What it costs if it is wrong

`substitute` is optional, and a rig without one behaves exactly as before. The
resolution path is only reached when gear fails to resolve, which used to be an
error, so nothing that built before can build differently now.
