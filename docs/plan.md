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
- [x] CLI entry point: `-f` flag for Procfile path, default `./Procfile`
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
- [x] Defer terminal restore at top of main (done in Phase 5a)

### Tests:
- [x] Exit code is printed: `[name] exited (code 0)`
- [x] Non-zero exit code is reported correctly
- [x] `Shutdown()` sends SIGTERM to all processes
- [x] `Shutdown()` waits for all goroutines to finish

---

## Phase 4 — Terminal Resize (SIGWINCH) `[x]`

**Files:** `gopty/manager.go` (extend), `gopty/manager_test.go` (extend), `cmd/main.go` (extend)

### Tests:
- [x] `ResizeAll` sets dimensions on PTY master

### Implement:
- [x] `Manager.ResizeAll(ws)` — calls `pty.Setsize` on each process's PTY master
- [x] `StartAll` sets initial PTY size by reading terminal size from `m.out` via `*os.File` type assertion
- [x] `listenResize` in `cmd/main.go` — SIGWINCH listener loops, reads fresh size via `pty.GetsizeFull`, calls `m.ResizeAll`

---

## Phase 5a — Process Selection Dialog & Attach `[x]`

**Goal:** Dialog modal on alternate screen for process selection, attach/detach via ctrl+].

**Files:** `gopty/dialog.go`, `gopty/dialog_test.go`, `gopty/controller.go`, `gopty/controller_test.go`, `gopty/utils.go`, `gopty/utils_test.go`, `gopty/manager.go` (extend), `gopty/manager_test.go` (extend), `cmd/main.go` (extend)

### Dialog (`dialog.go`):
- [x] `Dialog` struct with `Open()` — blocking, enters alt screen, runs input loop, returns `(int, bool)`
- [x] Arrow key navigation with clamping, Enter selects, Esc cancels
- [x] Renders process list with colors and reverse video highlight
- [x] Private `readKey()` using 3-byte buffer + string matching for ESC/arrow ambiguity
- [x] ANSI constants: `enterAltScreen`, `leaveAltScreen`, `showCursor`, `hideCursor`, `clearScreen`, `cursorHome`, `reverseVideo`

### Controller (`controller.go`):
- [x] State machine: Normal (all output) → Dialog → Attached
- [x] `Run()` loop checks `c.err == nil`, dispatches to `handleAllOut()` / `handleAttached()`
- [x] ctrl+c (byte 3) shuts down in normal mode
- [x] ctrl+] (byte 29) opens dialog in normal mode, detaches in attached mode
- [x] `Shutdown()` sets `c.err = io.EOF` and calls `m.Shutdown()`

### Utils (`utils.go`):
- [x] Shared byte constants: `byteCtrlC`, `byteCtrlBracket`, `byteEsc`, `byteEnter`
- [x] Shared sequence constants: `seqArrowUp`, `seqArrowDown`
- [x] `readBytes(r, n)` — shared by Dialog and Controller

### Manager changes (`manager.go`):
- [x] `Attach(index int) error` — index-based attachment with bounds checking
- [x] `Processes() []*Process` — exposes process list for Dialog

### Raw mode (`cmd/main.go`):
- [x] `rawMode(f)` enters raw mode, returns restore closure
- [x] Terminal enters raw mode at startup, `defer restore()`
- [x] Controller wired up: `go c.Run()`, `listenTerm(c.Shutdown)`
- [x] `\r\n` in process output for raw mode compatibility

### Tests:
- [x] Dialog: enter selects, arrow navigation with clamping, esc cancels
- [x] Controller: ctrl+c shutdown, ctrl+] dialog open/close, attach/detach flow
- [x] Manager: index-based attach, out-of-range errors
- [x] `readBytes`: reads up to n bytes, handles fewer available

---

## Phase 5b — Keystroke Forwarding `[x]`

**Goal:** Forward stdin to attached process's PTY master.

**Files:** `gopty/process.go` (extend), `gopty/process_test.go` (extend), `gopty/controller.go` (extend), `gopty/controller_test.go` (extend)

### Implement:
- [x] `Process.Write(buf)` method wrapping `p.master.Write(buf)`
- [x] In `handleAttached`, forward non-ctrl+] bytes via `c.manager.Attached().Write(buf)`
- [x] ctrl+c in attached mode forwarded to process (not go-pty shutdown)

### Tests:
- [x] `Process.Write` sends bytes to master
- [x] Controller forwards keystrokes (including ctrl+c) to attached process

---

## Final File Layout

```
go-pty/
  AGENTS.md
  README.md
  go.mod
  go.sum
  cmd/
    main.go                ← package main (CLI entry, signals, flags, raw mode)
  gopty/                   ← package gopty (all core logic)
      procfile.go
      procfile_test.go
      process.go
      process_test.go
      manager.go
      manager_test.go
      controller.go
      controller_test.go
      dialog.go
      dialog_test.go
      utils.go
      utils_test.go
  docs/
    spec.md
    plan.md
```

`cmd/main.go` is a thin entry point — flag parsing, signal wiring, raw mode, and calling into `gopty`. All logic and tests live in the `gopty` package.

---

## Testing Strategy

**Unit tests** — pure logic, no real PTYs:
- Procfile parsing
- Color assignment
- Output routing via `outputMode` callback
- Dialog navigation and selection (bytes.Reader as stdin)
- Controller state machine (bytes.Reader as stdin)
- `readBytes` utility

**Integration tests** — real PTYs with short-lived commands:
- Spawn `echo hello` in a PTY, verify prefixed output
- Spawn a process, verify exit code reporting
- Spawn a process, verify SIGTERM kills the process

**Test helpers:**
- `stubProcess(entry, mode, input)` — create Process with `os.Pipe` for testing `Read`
- `stubProcesses()` — create `[]*Process` for Dialog tests
- `stubManager()` — create Manager with single entry for Controller tests

---

## Risk Areas

| Risk | Mitigation |
|------|------------|
| Terminal left in raw mode on crash | `defer` restore at top of main |
| Zombie processes on unclean exit | SIGTERM to child processes on shutdown |
| Race between attach and process exit | `atomic.Pointer` for lock-free reads |
| Flaky integration tests with PTYs | Short-lived commands, timeouts |
