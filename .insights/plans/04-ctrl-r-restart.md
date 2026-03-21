# Phase 4: Ctrl+R restart

## Overview
Add ctrl+r keybinding to restart the currently attached process.

## Current State
- `process.go:51-73` — `Stream` defers `p.close()`, closes `terminated` at line 58, but defer runs after return
- `process.go:84-99` — `Shutdown` waits on `terminated` channel
- `manager.go:38-42` — `Stream` errors trigger `m.Shutdown()`, would fire during restart kill
- `controller.go:81-99` — `handleAttached` handles ctrl+] and ctrl+c, no ctrl+r
- `manager.go:108-116` — `attached()` returns `*Process`, no index tracking

## Out of Scope
- Restarting non-attached processes
- Custom restart timeout (use existing 5s shutdown timeout)

## Approach
- Move `p.close()` before `close(p.terminated)` in `Stream` (instead of defer). This ensures the old pty is closed before `Shutdown` returns, so `Restart` can safely call `Start()`.
- Add `restarting` atomic bool to `Process` so `Stream` returns nil during intentional restarts (prevents `m.Shutdown()` cascade).
- Add `RestartAttached()` to `Manager` so the controller doesn't need index tracking.

## Changes

- [ ] **`gopty/utils.go`** — Add `byteCtrlR = 18` constant

- [ ] **`gopty/process.go`** — Add restart support
  - Move `p.close()` from defer to inline, before `close(p.terminated)`
  - Add `restarting atomic.Bool` field
  - In `Stream`, if `p.restarting` is true, return nil instead of error
  - Add `Restart(timeout time.Duration) error`:
    1. Set `p.restarting.Store(true)`
    2. Call `Shutdown(timeout)` — when it returns, old pty is already closed
    3. Reset `p.terminated = make(chan struct{})`
    4. Reset `p.restarting.Store(false)`
    5. Call `Start()`

- [ ] **`gopty/manager.go`** — Add `RestartAttached() error`
  - Find attached process via `attached()`
  - Call `p.Restart(timeout)`
  - Re-launch `Stream` goroutine via `wg.Go`

- [ ] **`gopty/controller.go`** — Handle ctrl+r in `handleAttached()`
  - Add `byteCtrlR` case
  - Print `[go-pty] Restarting <name>...`
  - Call `c.manager.RestartAttached()`

- [ ] **`gopty/process_test.go`** — Add `TestProcess_Restart`
  - Start a process, restart it, verify new process is running
  - Verify old pty is closed, new pty is functional

## Success Criteria

### Automated:
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

### Manual:
- [ ] Attach to a process, press ctrl+r — process restarts, output resumes
- [ ] Other processes are unaffected during restart
- [ ] Pressing ctrl+r multiple times in succession doesn't crash
- [ ] A process crashing on its own still triggers `m.Shutdown()` as before
