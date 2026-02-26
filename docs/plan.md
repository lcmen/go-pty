# Development Plan (TDD)

Test-driven implementation. Each phase follows red-green-refactor: write failing tests first, then implement just enough to pass them.

**Status legend:** `[ ]` pending · `[~]` in progress · `[x]` done

---

## Phase 1 — Procfile Parsing `[ ]`

**Goal:** Parse a Procfile into process definitions.

**Files:** `gopty/procfile.go`, `gopty/procfile_test.go`

### Tests first (`procfile_test.go`):
- [ ] Parses `name: command` lines into `[]ProcEntry`
- [ ] Skips empty lines
- [ ] Skips `#` comment lines
- [ ] Handles whitespace around `:` separator
- [ ] Errors on lines with no `:` separator
- [ ] Errors on empty/missing file
- [ ] Handles commands containing `:` (only splits on first)

### Then implement (`procfile.go`):
- [ ] `ProcEntry` struct (`Name`, `Command`)
- [ ] `ParseProcfile(path string) ([]ProcEntry, error)`

### Then wire up (`cmd/main.go`):
- [ ] CLI entry point: accept optional path arg, default `./Procfile`
- [ ] Parse and print entries (sanity check, replaced in phase 2)

---

## Phase 2 — Process Spawning & Output Reading `[ ]`

**Goal:** Start commands in PTYs, read output with colored prefixes.

**Files:** `gopty/process.go`, `gopty/manager.go`, `gopty/manager_test.go`

### Tests first (`manager_test.go`):
- [ ] `StartAll` launches processes that run and produce output
- [ ] `readOutput` captures stdout from a simple `echo` command
- [ ] Output lines are prefixed with `[label]` and ANSI color codes
- [ ] Colors rotate through the palette for >6 processes
- [ ] Partial lines are buffered until `\n` arrives
- [ ] Multiple rapid lines from one process all get prefixed correctly

### Then implement:
- [ ] `Process` struct (label, cmd, master, color, lineBuf, dropped)
- [ ] `Manager` struct (processes, attached, rawState, mu, wg)
- [ ] `Manager.StartAll()` — spawn via `sh -c "exec <cmd>"`, `Setpgid`, `pty.Start`
- [ ] `readOutput` goroutine — read loop, mutex-guarded line extraction and printing
- [ ] Color assignment from palette with modulo wrapping

---

## Phase 3 — Process Exit & Shutdown `[ ]`

**Goal:** Clean termination, exit codes, signal handling.

**Files:** `gopty/manager.go` (extend), `gopty/manager_test.go` (extend)

### Tests first:
- [ ] Process exit flushes remaining partial line in buffer
- [ ] Exit code is printed: `[name] exited (code 0)`
- [ ] Non-zero exit code is reported correctly
- [ ] `Shutdown()` sends SIGTERM to all process groups
- [ ] `Shutdown()` waits for all goroutines to finish
- [ ] Terminal state is restored if in raw mode during shutdown

### Then implement:
- [ ] Exit handling in `readOutput` — flush lineBuf, print exit code
- [ ] `Manager.Shutdown()` — `Kill(-pid, SIGTERM)` per process, `wg.Wait()`
- [ ] Signal listener for SIGINT/SIGTERM in `cmd/main.go`
- [ ] Defer terminal restore at top of main

---

## Phase 4 — Terminal Resize (SIGWINCH) `[ ]`

**Files:** `gopty/manager.go` (extend), `gopty/manager_test.go` (extend)

### Tests first:
- [ ] Initial PTY size matches a provided size
- [ ] `ResizeAll` propagates dimensions to all PTY masters

### Then implement:
- [ ] Set initial PTY size on startup
- [ ] SIGWINCH listener calls `pty.InheritSize(os.Stdin, p.master)` for all processes

---

## Phase 5 — Step-In Command Parsing `[ ]`

**Goal:** Parse `!N` and `!name` from stdin input.

**Files:** `gopty/input.go`, `gopty/input_test.go`

### Tests first (`input_test.go`):
- [ ] `!1` returns process index 0
- [ ] `!3` returns process index 2
- [ ] `!0` returns error (1-indexed)
- [ ] `!99` returns error for out-of-range
- [ ] `!web` matches process by label (case-insensitive)
- [ ] `!Web` matches `web` (case-insensitive)
- [ ] `!nonexistent` returns error
- [ ] Leading/trailing whitespace is trimmed
- [ ] Input without `!` prefix is ignored

### Then implement (`input.go`):
- [ ] `ParseStepInCommand(input string, processes []*Process) (*Process, error)`

---

## Phase 6 — Stepped-In Mode `[ ]`

**Goal:** Raw terminal forwarding to one process, ctrl+] to detach.

**Files:** `gopty/manager.go` (extend), `gopty/manager_test.go` (extend)

### Tests first:
- [ ] `stepIn` sets `m.attached` to the target process
- [ ] While attached, `readOutput` writes raw output for the attached process (no prefix)
- [ ] While attached, other processes' output goes to `lineBuf` only (not stdout)
- [ ] Byte value 29 (ctrl+]) triggers `stepOut`
- [ ] `stepOut` sets `m.attached` to nil
- [ ] `stepOut` flushes buffered output from all processes with prefixes
- [ ] Non-ctrl+] bytes are written to `p.master`

### Then implement:
- [ ] `stepIn(p)` — print instructions, `term.MakeRaw`, set attached
- [ ] `stepOut()` — restore terminal, clear attached, flush buffers
- [ ] Modify `handleStdin` for raw mode byte forwarding
- [ ] Modify `readOutput` to check attached state and route output accordingly

---

## Phase 7 — Buffer Overflow `[ ]`

**Goal:** Cap background buffers at 1MB, track dropped lines.

**Files:** `gopty/manager.go` (extend), `gopty/manager_test.go` (extend)

### Tests first:
- [ ] Buffer exceeding 1MB is trimmed from the front
- [ ] Trimming happens at a line boundary (next `\n` after cut point)
- [ ] `p.dropped` count reflects number of lines removed
- [ ] On flush after step-out, drop count message is printed before buffered content
- [ ] `p.dropped` resets to 0 after flush
- [ ] Buffer exactly at 1MB is not trimmed

### Then implement:
- [ ] Overflow check in `readOutput` buffering path
- [ ] Line-boundary trimming logic
- [ ] Drop count message in `stepOut` flush

---

## Final File Layout

```
go-pty/
  AGENTS.md
  README.md
  go.mod
  go.sum
  cmd/
    main.go                ← package main (CLI entry, signals, flags)
  gopty/                   ← package gopty (all core logic)
      procfile.go
      procfile_test.go
      process.go
      manager.go
      manager_test.go
      input.go
      input_test.go
  docs/
    spec.md
    plan.md
    concepts.md
```

`cmd/main.go` is a thin entry point — flag parsing, signal wiring, and calling into `gopty`. All logic and tests live in the `gopty` package.

---

## Testing Strategy

**Unit tests** — pure logic, no real PTYs:
- Procfile parsing
- Command parsing (`!N`, `!name`)
- Buffer overflow / trimming logic
- Color assignment

**Integration tests** — real PTYs with short-lived commands:
- Spawn `echo hello` in a PTY, verify prefixed output
- Spawn a process, verify exit code reporting
- Spawn a process, verify SIGTERM kills the process group
- Step-in/step-out with a process that echoes stdin

**Test helpers:**
- `captureOutput(fn)` — redirect stdout to a buffer, run fn, return captured string
- `testManager(entries)` — create a Manager with test ProcEntries without starting processes
- `waitFor(condition, timeout)` — poll for async conditions in goroutine tests

---

## Risk Areas

| Risk | Mitigation |
|------|------------|
| Terminal left in raw mode on crash | `defer` restore at top of main |
| Zombie processes on unclean exit | Process group kill reaches descendants |
| Mutex held during blocking I/O | Read outside lock; only processing locked |
| Race between step-in and process exit | Check attached under mutex |
| Flaky integration tests with PTYs | Short-lived commands, `waitFor` with timeouts |
