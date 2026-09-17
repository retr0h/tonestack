# Add evidence from a recording

A rig gets more trustworthy by having its claims made checkable, not by someone
asserting them more confidently.

```yaml
- role: amp
  gear: Ampeg SVT
  evidence:
    - { kind: cited, url: "…", note: "Bass Player interview, 2004" }
    - { kind: video, url: "…", at: "1:42", note: "SVT visible on stage" }
  confidence: high
```

Evidence attaches to a claim, not to the document, because the amp may come from
an interview and a drive figure from measuring a corpus.

**A URL does not make a claim true.** It makes it checkable, which is what lets
somebody correct one line instead of re-deriving a rig.

Audio evidence carries measurements rather than conclusions, and that
distinction is load-bearing: measuring a record measures the bass, the player,
the amp, the mic, the desk, the master and the encoder. Those numbers compare
against the same measurement taken from a generated preset, giving a direction.
They are not knob positions. The `caveat` field is where a rig says which of the
two it means.
