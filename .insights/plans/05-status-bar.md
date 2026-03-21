# Phase 5: Status bar

## Overview
Add a persistent status bar on the last terminal line showing all process names, highlighting the attached one.

## Current State
- `controller.go:10-25` — Controller owns stdin/stdout and knows about attach/detach
- `manager.go:67-69` — `Processes()` returns process list
- `cmd/main.go:71-88` — `listenResize` calls `m.ResizeAll`; needs to also update status bar
- All processes are always running (all-or-nothing shutdown), so state can be hardcoded

## Out of Scope
- Per-process state tracking (running/exited/restarting)
- Clickable or interactive status bar

## Approach
- Create `StatusBar` struct in `gopty/statusbar.go`, owned by Controller
- Uses ANSI scroll region to reserve the last terminal line
- Renders process names with the attached one highlighted
- Controller calls `StatusBar.Update` on attach/detach
- `StatusBar.Resize` wired into `listenResize` via Controller or Manager

## Changes

- [ ] **`gopty/statusbar.go`** (new) — `StatusBar` struct
  - `NewStatusBar(stdout io.Writer, rows int) *StatusBar`
  - On init, set scroll region `\033[1;{rows-1}r` reserving last line
  - `Update(processes []*Process, attached *Process)` — render process names on the last line using `\033[{rows};1H`, highlight attached process (e.g. reverse video), all others in their color
  - `Resize(rows int)` — recalculate scroll region, re-render
  - `Close()` — clear scroll region (`\033[r`), erase status line

- [ ] **`gopty/controller.go`** — Wire StatusBar
  - Add `statusBar *StatusBar` field to Controller
  - Init StatusBar in `NewController` (detect terminal rows)
  - Call `statusBar.Update` after attach and detach
  - Call `statusBar.Close` in cleanup path

- [ ] **`cmd/main.go`** — Wire resize
  - `listenResize` handler calls both `m.ResizeAll` and status bar resize
  - Pass terminal rows to Controller or StatusBar on SIGWINCH

- [ ] **`gopty/statusbar_test.go`** (new) — Test StatusBar rendering
  - Verify scroll region escape sequences on init
  - Verify Update output with and without an attached process
  - Verify Close clears scroll region

## Success Criteria

### Automated:
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

### Manual:
- [ ] Status bar appears on the last line showing all process names
- [ ] Attached process is visually highlighted
- [ ] Resizing terminal updates the status bar position
- [ ] Status bar cleans up on exit (no leftover scroll region)
