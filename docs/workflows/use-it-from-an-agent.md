# Use it from an agent

`tonestack mcp start` gives an agent the same operations as tools, with typed
results instead of text to parse. Claude Code starts it for you once it is
added:

```bash
claude mcp add tonestack -- tonestack mcp start
```

That offers everything except writing to a pedal. To let the agent import, copy
and swap slots, add it with `tonestack mcp start --allow-writes` instead. Each
write still saves what it replaces to a file first. Selecting a slot works
either way.

The same flag decides whether `preset_build` and `preset_export` may write over
a file already at the path the agent names. Without it they refuse, and the
refusal is part of the write: a file that appears while a build or an export is
running is kept too. With it they replace the file, as `presets make` and
`presets export` always do.

The agent sees the rigs `recipes list` shows, your own recipes beside the ones
that ship. `rigs_list`, `rig_show` and `preset_build` reach a rig you wrote with
`recipes new` as soon as the file is there.

Quit HX Edit before asking for anything that reaches the pedal.
