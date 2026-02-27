# Development Plan (TDD)

Test-driven implementation. Each phase follows red-green-refactor: write failing tests first, then implement just enough to pass them.

**Status legend:** `[ ]` pending · `[~]` in progress · `[x]` done

---

## Phase 1 — Procfile Parsing `[x]`

**Goal:** Parse a Procfile into process definitions.

**Files:** `gopty/procfile.go`, `gopty/procfile_test.go`

### Tests first (`procfile_test.go`):
- [x] Parses `name: command` lines into `[]Entry`
- [x] Skips empty lines
- [x] Skips `#` comment lines
- [x] Handles whitespace around `:` separator
- [x] Errors on lines with no `:` separator
- [x] Errors on empty/missing file
- [x] Handles commands containing `:` (only splits on first)

### Then implement (`procfile.go`):
- [x] `Entry` struct (`Name`, `Command`)
- [x] `File` struct (stores lines and path)
- [x] `Open(path string) (*File, error)` — reads file into `File` struct
- [x] `File.Parse() ([]Entry, error)` — parses lines into entries
- [x] Helper functions: `isComment`, `isEmpty`, `extractNameAndCommand`

### Then wire up (`cmd/main.go`):
- [x] CLI entry point: accept optional path arg, default `./Procfile`
- [x] Parse and print entries (sanity check, replaced in phase 2)

---

## Phase 2 — Process Spawning & Output Reading `[x]`

**Goal:** Start commands in PTYs, read output with colored prefixes.

**Files:** `gopty/process.go`, `gopty/process_test.go`, `gopty/manager.go`, `gopty/manager_test.go`

### Tests first:
- [x] `NewManager` creates Process structs with correct entries and colors
- [x] `Process.Read` with `OutputAll` prefixes lines with colored `[name]`
- [x] `Process.Read` with `OutputAttached` prints raw lines
- [x] `Process.Read` with `OutputIgnored` drops all output
- [x] Colors rotate through the 12-color palette

### Then implement:
- [x] `Process` struct (Entry, Color, cmd, master, outputMode)
- [x] `Manager` struct (processes, attached `atomic.Pointer[Process]`, wg)
- [x] `NewProcess(entry, index)` — assigns color from palette
- [x] `Process.Start()` — spawn via `sh -c "exec <cmd>"`, `pty.Start`
- [x] `Process.Read(w)` — scan loop, routes via `outputMode` callback
- [x] `OutputMode` enum (`OutputAll`, `OutputAttached`, `OutputIgnored`)
- [x] `Manager.outputMode(p)` — factory returns closure capturing `m.attached` and `p`
- [x] `Manager.StartAll()` — start processes, launch read goroutines via `wg.Go`
- [x] `Manager.Attach(name)` / `Manager.Detach()` / `Manager.Wait()`
- [x] Wire up in `cmd/main.go` with error handling

---

## Phase 3 — Process Exit & Shutdown `[x]`

**Goal:** Clean termination, exit codes, signal handling.

**Files:** `gopty/process.go` (extend), `gopty/process_test.go` (extend), `gopty/manager.go` (extend), `gopty/manager_test.go` (extend), `cmd/main.go` (extend)

### Implement:
- [x] Exit handling in `Read` — print exit code when process ends
- [x] `Manager.Shutdown()` — send SIGTERM to each process, `wg.Wait()`
- [x] Signal listener for SIGINT/SIGTERM in `cmd/main.go`
- [ ] Defer terminal restore at top of main (deferred to Phase 6)

### Tests:
- [x] Exit code is printed: `[name] exited (code 0)`
- [x] Non-zero exit code is reported correctly
- [x] `Shutdown()` sends SIGTERM to all processes
- [x] `Shutdown()` waits for all goroutines to finish

---

## Phase 4 — Terminal Resize (SIGWINCH) `[ ]`

**Files:** `gopty/manager.go` (extend)

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
- [ ] `Attach` sets `m.attached` to the target process
- [ ] While attached, output is raw (no prefix) for attached process
- [ ] While attached, other processes' output is dropped
- [ ] Byte value 29 (ctrl+]) triggers `Detach`
- [ ] `Detach` sets `m.attached` to nil
- [ ] Non-ctrl+] bytes are written to `p.master`

### Then implement:
- [ ] `Attach(name)` — print instructions, `term.MakeRaw`, set attached
- [ ] `Detach()` — restore terminal, clear attached
- [ ] Modify stdin handler for raw mode byte forwarding

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
      process_test.go
      manager.go
      manager_test.go
      input.go
      input_test.go
  docs/
    spec.md
    plan.md
```

`cmd/main.go` is a thin entry point — flag parsing, signal wiring, and calling into `gopty`. All logic and tests live in the `gopty` package.

---

## Testing Strategy

**Unit tests** — pure logic, no real PTYs:
- Procfile parsing
- Command parsing (`!N`, `!name`)
- Color assignment
- Output routing via `outputMode` callback

**Integration tests** — real PTYs with short-lived commands:
- Spawn `echo hello` in a PTY, verify prefixed output
- Spawn a process, verify exit code reporting
- Spawn a process, verify SIGTERM kills the process
- Step-in/step-out with a process that echoes stdin

**Test helpers:**
- `stubProcess(entry, mode, input)` — create Process with `os.Pipe` for testing `Read`

---

## Risk Areas

| Risk | Mitigation |
|------|------------|
| Terminal left in raw mode on crash | `defer` restore at top of main |
| Zombie processes on unclean exit | SIGTERM to child processes on shutdown |
| Race between attach and process exit | `atomic.Pointer` for lock-free reads |
| Flaky integration tests with PTYs | Short-lived commands, timeouts |
