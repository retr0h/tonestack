# For agents

Three rules beyond the workflows above.

**Run the source, not a release.** The pages write commands as `tonestack ...`.
From a checkout that is `go run main.go ...`, which builds what is in front of
you. An installed `tonestack` is a release and will not carry a command added on
the branch you are reading.

**Say which claim you have.** "The rig validates against the catalog", "HX Edit
imported the file" and "the hardware loaded it" are three different claims, and
only the first is currently possible here. Do not report one as another.

**Never invent a model identifier.** If `catalog list --search` does not find
the gear, the device does not model it. Say so and suggest what it does have,
rather than writing an identifier that looks plausible.
