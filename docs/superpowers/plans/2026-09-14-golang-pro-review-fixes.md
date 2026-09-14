# golang-pro review fixes: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers-extended-cc:subagent-driven-development to implement this plan
> task by task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** fix every bug, test gap and stale document the golang-pro review of
tonestack found. The exception is the SDK redesign, which is Track B (task 49).

**Architecture:** four tasks, each one area of the code, on one branch
(`fix/golang-pro-review`) and one pull request. Every fix starts from a test
that fails before the change.

**Tech Stack:** Go 1.27, testify/suite, go.uber.org/mock, cobra, go-sdk v1.7.0.

**Spec:** the review itself, recorded in task 48, with the rulings below. Track
B (task 49) is out of scope: finding 4 (a reader goroutine), finding 10 (SDK
options and sessions), the architecture changes, and the 494 one-line
signatures.

## Global Constraints

**How to work, for speed.**

- Iterate with per-package commands only:
  - `mise exec -- go test -cover ./<pkg>/`
  - `mise exec -- go test -run 'Suite/TestX' ./<pkg>/`
  - `mise exec -- go test -coverprofile=/tmp/c.out ./<pkg>/ && mise exec -- go tool cover -func=/tmp/c.out | grep -v 100.0%`
- Use `-race` only on `pkg/sdk/internal/device` and `pkg/mcp/...`.
- Do NOT run `just ready` or `just test` inside a task. The controller runs both
  once before the PR.

**Rules.**

- Coverage gate 99%. Never edit `.github/codecov.yml`, the justfile coverage
  target or `.coverignore`. Each touched package keeps its coverage (100% where
  it is 100% today; `device` must not drop below its current 90.6%).
- Interfaces live where they are consumed. Test doubles are mockgen only, never
  hand-written.
- `*_public_test.go` in the `_test` package, `*_test.go` for internal tests.
  testify suites, table-driven, one suite method per function under test.
- MIT header on every new `.go` file. Multi-line signatures on every function
  you add or change.
- Never set `SilenceUsage`. No hardware: no `-tags device` and no device
  commands without `--file`.
- Commit trailer, exactly:
  `🤖 Generated with [Claude Code](https://claude.ai/code)`, a blank line, then
  `Co-Authored-By: Claude <noreply@anthropic.com>`.
- Stage files explicitly and do not push.

**User decisions (already made):**

- "lets fix all of those": every Track A finding gets fixed.
- "i dont think you should split it up": the review was holistic; the fixes
  still go through task-sized pieces.
- Optimise for development speed: per-package tests while working, the full gate
  once before the PR.

## Rulings on points the review left open

01. **`ErrNoUSBSupport` stays exported on darwin.** Callers on every OS match it
    by one name; removing it on darwin would break builds that compile an
    `errors.Is` against it. No change.
02. **A write that has started finishes.** Once the first chunk of a message has
    gone out, `write` finishes the message and waits for its reply under
    `context.WithoutCancel`, bounded by `commitBudget`. A write that completes
    returns nil; the next operation sees the cancellation.
03. **Backups.**
    - Named `<label>-s<setlist>-<UTC 20060102-150405.000000000>.hlx`, written
      with no-clobber semantics, so a collision fails rather than overwrites.
    - A body that translates to no blocks, but is not empty, is kept as the raw
      device bytes in a `.bin` file of the same name, so nothing a device held
      is lost.
04. **`ExportWith` as hlx with no document** returns `ErrEmptySlot` and writes
    nothing.
05. **Atomic writes** go through a new internal package,
    `pkg/sdk/internal/atomicfile`:
    - `Write(path string, data []byte, perm os.FileMode) error`: temp file in
      the same directory, then rename.
    - `WriteNew(path string, data []byte, perm os.FileMode) error`: temp file,
      then `os.Link`. Fails with an error wrapping `fs.ErrExist` if the path
      exists. The temp file is always removed.
06. **MCP.** `preset_build` and `preset_export` refuse to write over an existing
    `out` unless the server runs with `--allow-writes`. The error names the path
    and the flag.
07. **`recipes.New` with an empty `Dir`** returns a new `ErrNoDir`. Update
    `sdk.NewRecipe.Dir`'s comment. `cmd/recipes_new.go` already fills it in.
08. **`wire.Response.Err`.** A status other than done (0), accepted (1) or
    refused (255) is an error (`UnexpectedStatusError`).
09. **`findDevice`.** It matches vendor and product. An enumeration error closes
    whatever was returned and fails, even if something was found. The existing
    row "one that complained and found something anyway" flips to expect the
    error.
10. **`Close`.** It runs under
    `context.WithTimeout(context.Background(), closeBudget)`, with
    `var closeBudget = 10 * time.Second`, exported to tests.

______________________________________________________________________

### Task 1: The device layer (Critical 1, Important 3, device minors)

**Goal:** a cancelled write can no longer burst chunks at the pedal, a real read
error ends a wait immediately with that error, and `Close` and `findDevice`
behave as the rulings say. Device tests get faster.

**Files:**

- Modify:
  - `pkg/sdk/internal/device/transport.go` (receive, drain)
  - `pkg/sdk/internal/device/handshake.go` (handshake, openService, awaitReply)
  - `pkg/sdk/internal/device/write.go` (write, stream)
  - `pkg/sdk/internal/device/conversation.go` (Close, closeBudget, firstSeq
    comment, usb_open.go reference)
  - `pkg/sdk/internal/device/discover_bus.go` (findDevice, usb.go reference)
  - `pkg/sdk/internal/device/types.go` (package comment says "Package sdk")
  - `pkg/sdk/internal/device/export_test.go`
- Test:
  - `transport_public_test.go`
  - `handshake_public_test.go`
  - `write_public_test.go`
  - `conversation_public_test.go`
  - `discover_bus_public_test.go`

**Acceptance Criteria:**

- [ ] **`receive` returns `(bool, error)`.**
  - It is quiet (`false, nil`) when the read returned nothing with no error, or
    `context.DeadlineExceeded` while the caller's ctx is still live.
  - It returns `ctx.Err()` when the caller's ctx ended.
  - It returns the read error wrapped (`reading from the device: %w`) for
    anything else.
  - Every caller handles the error: `awaitReply` and `stream` return it,
    `handshake` and `openService` return it, and `drain` stops on it.
- [ ] **`TestCall` has a row for a device whose read fails with a non-timeout
  error.**
  - The call returns that error, wrapped, within one read (well under
    `replyBudget`).
  - The message is not "no reply".
- [ ] **`write` checks `ctx` before the first chunk.** A ctx already cancelled
  sends nothing and returns `context.Canceled`.
- [ ] **`TestWritePreset` has a row where ctx is cancelled after the first chunk
  is written.** Every chunk is still sent, the reply is still awaited, and the
  write returns nil. The scripted writer cancels on its first Write.
- [ ] **`Close` returns within `closeBudget` against a receiver that never goes
  quiet.** `TestClose` has a row for this, with `CloseBudget` shortened.
- [ ] **`findDevice` matches vendor and product.**
  - `TestOpenOver` has a row for a device with a Helix product ID and a foreign
    vendor, which is not claimed.
  - The enumeration-error row now expects the error, and every returned handle
    is closed.
- [ ] **Stale comments fixed:**
  - `types.go`: "Package device talks to…".
  - `conversation.go:23` names `usb_darwin.go`.
  - `discover_bus.go` "usb.go" becomes `usb_darwin.go`.
  - The `firstSeq` comment reads "Two, not one".
- [ ] **Test budgets shortened.**
  - `drainBudget` becomes a var exported as `DrainBudget`.
  - A `TestMain` in the `device_test` package shortens `ReplyBudget`,
    `DrainBudget`, `CommitBudget`, `FlashBudget` and `CloseBudget` for the whole
    package, except where a test sets its own.
  - `go test -race ./pkg/sdk/internal/device/` runs in under 15s.
- [ ] Device package coverage is at least its current 90.6%.

**Verify:** `mise exec -- go test -race -cover ./pkg/sdk/internal/device/` → ok,
coverage ≥ 90.6%, under 15s

**Key code:**

```go
// receive reads one transfer and routes every frame in it.
func (s *session) receive(
	ctx context.Context,
	wait time.Duration,
) (bool, error) {
	rctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	buf := make([]byte, readBuffer)

	n, err := s.in.ReadContext(rctx, buf)
	switch {
	case err == nil:
	case ctx.Err() != nil:
		return false, ctx.Err()
	case errors.Is(err, context.DeadlineExceeded):
		// The ordinary case: the device had nothing to say in time.
		return false, nil
	default:
		return false, fmt.Errorf("reading from the device: %w", err)
	}
	// ... frame routing unchanged, return got, nil
}
```

```go
func (s *session) write(
	ctx context.Context,
	opcode uint64,
	args []wire.Arg,
) error {
	// Before anything is sent. Afterwards the message is finished whatever
	// happens: a device fed half a message and then a burst is the stall
	// docs/protocol.md describes.
	if err := ctx.Err(); err != nil {
		return err
	}

	c, ok := s.chans[channelData]
	// ...
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commitBudget)
	defer cancel()

	if err := s.stream(wctx, c, body); err != nil {
		return err
	}

	if _, err := s.awaitReply(wctx, c, txn, opcode); err != nil {
		return err
	}

	settle()

	return nil
}
```

______________________________________________________________________

### Task 2: Slots, presets and recipes, data safety (Critical 2, Important 5, 6, 7, recipes)

**Goal:** no preset or file can be silently lost. Cross-setlist edits carry the
right names, backups never collide and keep the preset's name, encoder errors
are reported, and user files are written atomically.

**Files:**

- Create:
  - `pkg/sdk/internal/atomicfile/atomicfile.go`
  - `pkg/sdk/internal/atomicfile/atomicfile_public_test.go`
- Modify:
  - `pkg/sdk/internal/slots/editdevice.go` (names, swapTwo, copyOne)
  - `pkg/sdk/internal/slots/backup.go` (at, keep, backup, replacing)
  - `pkg/sdk/internal/slots/importdevice.go` (replacing call)
  - `pkg/sdk/internal/slots/transfer.go` (write)
  - `pkg/sdk/internal/slots/open.go` (save, and the package doc, which must
    describe both file and device flows)
  - `pkg/sdk/internal/slots/compile.go`
  - `pkg/sdk/internal/presets/make.go` (openStats error, write)
  - `pkg/sdk/internal/recipes/new.go` (ErrNoDir, WriteNew)
  - `pkg/sdk/client.go` (`NewRecipe.Dir` comment)
- Test:
  - `slots/editdevice_public_test.go`
  - `slots/backup_test.go`
  - `slots/device_public_test.go`
  - `slots/edit_public_test.go`
  - `presets/make_public_test.go`
  - `recipes/new_public_test.go`

**Acceptance Criteria:**

- [ ] **`atomicfile.Write` and `atomicfile.WriteNew` exist as in ruling 5.**
  - Tests cover: a new file, overwriting (Write), refusing an existing file
    (WriteNew, `errors.Is(err, fs.ErrExist)`), an unwritable directory, and no
    temp file left behind on every path.
- [ ] **`names()` lists `FromSetlist` and, when different, `ToSetlist`, and
  looks each slot up in its own setlist.**
  - `TestSwapWith` and `TestCopyWith` have a cross-setlist row (from 0/01A to
    1/01A).
  - Each asserts the names written by `WriteNamedPreset` and `Change.Replaced`.
- [ ] **`at` carries `setlist` and `name`, and `keep` passes both into
  `DeviceOptions`.**
  - Backup names follow ruling 3, and the preset's real name reaches the `.hlx`.
  - `replacing` takes the name; copy and import pass the destination's listed
    name.
- [ ] **Two backups of the same slot and setlist, taken in the same second, are
  two files.** `TestKeep` has a row for this.
- [ ] **A non-empty body that translates to no blocks is kept as `.bin` raw
  bytes.** `TestBackup` has a row using the existing `emptied()` fixture.
- [ ] **`write` in `transfer.go` checks the result.**
  - As hlx with `read.Doc == nil`, it returns `ErrEmptySlot`.
  - `preset.Write` and `rig.Write` errors are returned.
  - The file is written with `atomicfile.Write`.
  - `ExportWith` has a row for a slot with no blocks.
- [ ] **`save` returns `setlist.Write`'s error and uses `atomicfile.Write`.**
  - `compile.go` and `presets/make.go` return `preset.Write`'s error and use
    `atomicfile.Write`.
  - Backups use `atomicfile.WriteNew`.
- [ ] **`Make` with a non-empty `StatsPath` that cannot be opened returns the
  error.** The "no statistics to be had" row flips to expect an error containing
  the path.
- [ ] **`recipes.New` checks its inputs and never overwrites.**
  - An empty `Dir` returns `ErrNoDir`.
  - The file is written with `atomicfile.WriteNew`.
  - An existing file returns `*ExistsError`, mapped from `fs.ErrExist`; the
    `os.Stat` pre-check is removed.
- [ ] `atomicfile`, `slots`, `presets` and `recipes` all stay at 100% coverage.

**Verify:**
`mise exec -- go test -cover ./pkg/sdk/internal/atomicfile/ ./pkg/sdk/internal/slots/ ./pkg/sdk/internal/presets/ ./pkg/sdk/internal/recipes/ ./pkg/sdk/`
→ all ok, the four internal packages at 100.0%

**Key code:**

```go
// WriteNew puts data at path only if nothing is there yet.
//
// The whole file appears at once or not at all, and an existing file is left
// alone: os.Link refuses to replace its target.
func WriteNew(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	tmp, err := temp(path, data, perm)
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp) }()

	if err := os.Link(tmp, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
```

______________________________________________________________________

### Task 3: Preset, wire and MCP (Important 8, 9, wire minors, MCP tests)

**Goal:**

- Compiling into a template clears blocks on processors the rig does not use.
- The wire decoder matches keys of any width and reports unknown statuses.
- Dead code goes.
- MCP cannot overwrite files without `--allow-writes`.
- The two timing-based MCP tests become deterministic.

**Files:**

- Modify:
  - `pkg/sdk/preset/write.go` (SetSpec)
  - `pkg/sdk/preset/types.go:53` (the Rest comment, which must say only the
    top-level `meta` is kept raw)
  - `pkg/sdk/internal/wire/rpc.go` (DecodeResponse, Err, UnexpectedStatusError)
  - `pkg/sdk/internal/wire/preset.go` and `place.go` (keyBypassed → keyEnabled)
  - `pkg/sdk/internal/wire/splice.go` and `place.go` (delete `splice`,
    `spliceRaw`, `open`)
  - `pkg/sdk/internal/wire/encode.go` (encodeString)
  - `pkg/sdk/internal/wire/export_test.go`
  - `pkg/mcp/internal/tools/register.go`, `offline.go`, `device.go`, `errors.go`
- Test:
  - `preset/write_public_test.go`
  - `wire/rpc_public_test.go`
  - `wire/splice_public_test.go` and `wire/place_public_test.go` (remove the
    tests of deleted functions)
  - `mcp/internal/tools/offline_public_test.go`, `device_public_test.go`
  - `pkg/mcp/stdio_public_test.go`

**Acceptance Criteria:**

- [ ] **`SetSpec` removes block keys from every `dsp*` tone entry before placing
  the new chain.** `TestSetSpec` has a row: a document with blocks on `dsp0` and
  `dsp1` and a chain on `dsp0` only leaves `dsp1` with no block keys and its
  routing keys intact.
- [ ] **`DecodeResponse` normalises every map key with `asUint`.**
  `TestDecodeResponse` has a row with the txn, status and result keys encoded as
  uint16.
- [ ] **`Response.Err` returns `*UnexpectedStatusError` for any status not 0, 1
  or 255.** Rows cover 0, 1, 255 and 7.
- [ ] **The key is named for what it holds:** `keyBypassed` is renamed
  `keyEnabled` everywhere.
- [ ] **Dead code is gone.**
  - `splice`, `spliceRaw` and `open` are deleted, along with their export_test
    aliases and tests.
  - Anything else they leave unreachable is deleted too: run
    `mise exec -- go run golang.org/x/tools/cmd/deadcode@latest -test ./pkg/sdk/internal/wire/`.
  - If `encodeString` is still reachable (through `place.go`'s `encodeLike`),
    its cases check the original's width first. A `str16` original of 32–255
    bytes stays `str16`, with a test row for it.
- [ ] **MCP no longer overwrites existing files without `--allow-writes`.**
  - `handlers` holds `allowWrites`.
  - `preset_build` and `preset_export` return `ErrWouldOverwrite` wrapped with
    the path when `out` exists and writes are not allowed.
  - Rows cover: new path (ok), existing path with writes off (IsError, client
    not called), existing path with writes on (ok).
- [ ] **The MCP lock test uses a barrier, not a sleep.**
  - The first `Devices` call blocks on a channel after signalling it entered.
  - The second call is started, and the test asserts it has not entered within
    200ms.
  - The channel is then closed, and both complete.
- [ ] **The stdio hang-up test is deterministic.**
  - The reader returns the initialize line, then signals, then returns EOF.
  - The writer waits for that signal before writing.
  - It passes 20 runs with `-race -count=20`.
- [ ] `preset`, `wire`, `pkg/mcp` and `pkg/mcp/internal/tools` stay at 100%
  coverage.

**Verify:**
`mise exec -- go test -cover ./pkg/sdk/preset/ ./pkg/sdk/internal/wire/ && mise exec -- go test -race -count=20 -cover ./pkg/mcp/...`
→ all ok at 100.0%

______________________________________________________________________

### Task 4: Test doubles and documents (the review's test and doc findings)

**Goal:** no hand-written doubles remain for interfaces this project defines,
and the stale documents say what the code does.

**Files:**

- Create: `pkg/sdk/internal/device/generate_test.go`, or add directives to the
  existing `generate.go`. Its mockgen directives write `*.gen_test.go` mocks for
  the unexported `sender`, `receiver`, `bus`, `handle` and `endpoints`
  interfaces, in package `device`, as CONTRIBUTING describes for unexported
  interfaces.
- Modify:
  - `pkg/sdk/internal/device/device_public_test.go` (scripted)
  - `discover_bus_public_test.go` (fakeBus, fakeHandle, fakeEnds)
  - `export_test.go`
  - `pkg/sdk/client_public_test.go` (bus → `mocks.MockBus`)
  - `pkg/sdk/internal/attached/list_public_test.go` (lister → `mocks.MockBus`)
  - `docs/protocol.md:93` and `:190`

**Acceptance Criteria:**

- [ ] **No hand-written doubles remain.**
  `grep -rn "^type \(scripted\|fakeBus\|fakeHandle\|fakeEnds\|bus\|lister\) struct" pkg/sdk --include='*_test.go'`
  prints nothing.
- [ ] **`scripted` is replaced by a constructor over the generated mocks.** It
  returns the sender and receiver mocks plus a pointer to the recorded writes;
  the reply queue lives in a closure. Call sites keep their shape as far as
  possible (`answers(frames...)`). When the queue is empty, the receiver returns
  `context.DeadlineExceeded`, as IOKit does after `readUntil`.
- [ ] **`docs/protocol.md` matches the code.**
  - Line 93 says `pkg/sdk/internal/wire`.
  - Lines 190–191 say that only `pkg/sdk/internal/device/usb_darwin.go` needs
    hardware, that it is counted against the 99% gate, and that
    `pkg/sdk/internal/wire` needs none.
  - The change goes through the unslop skill.
- [ ] `sdk`, `attached` and `device` coverage is unchanged, or higher.

**Verify:**
`mise exec -- go test -race -cover ./pkg/sdk/internal/device/ ./pkg/sdk/internal/attached/ ./pkg/sdk/`
→ all ok, coverage unchanged

______________________________________________________________________

## After the four tasks

The controller:

1. Runs `mise exec -- just ready` and `mise exec -- just test` once, and fixes
   whatever they report.
2. Runs one whole-branch review.
3. Opens one PR. The description lists each review finding, its fix, the rulings
   above, and which claims were verified. None were verified on hardware.
