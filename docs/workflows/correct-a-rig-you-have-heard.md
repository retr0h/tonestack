# Correct a rig you have heard

This is the loop that matters, because **nothing here can hear**. Every other
input is a measurement or an assertion; you are the only thing that can say
whether it sounds right.

1. Play it.
2. Say what is wrong in your own words. "Too clunky", "the drive is muddy".
3. Change the rig.
4. **Record what you asked for, what changed, and what you thought of it.**

That last step is the one people skip and the only one that cannot be recovered
later. A catalog can be regenerated next year; nobody can go back and ask
themselves what they thought of round three.

```yaml
mutations:
  - ask: make it clunkier
    changed:
      - { path: chain[1].settings.drive, from: 0.47, to: 0.58 }
    reason: >-
      Line 6 document Sag as "lower values offer tighter responsiveness…
      higher values provide more touch dynamics & sustain". Read "clunky" as
      a looser power-amp feel rather than more gain.
    verdict: closer, but muddy now, so keep the feel and put the drive back
```

`reason` is where an agent records its *interpretation*, cited. If it read
"clunkier" as drive when you meant sag, that line is what shows the reading was
wrong rather than only the value.

If several rounds of settings changes all come back negative, stop turning
knobs. Repeated failure at that layer is evidence against something higher up,
usually the amp.
