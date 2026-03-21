# Phase 2: Global env file (`-e`)

## Overview
Add a `-e` flag to load a `.env` file and pass its variables to all child processes.

## Current State
- `NewProcess(entry, index)` at `process.go:29` — no env support
- `NewManager(entries, stdout)` at `manager.go:20` — no env support
- `Process.Start()` at `process.go:37` — doesn't set `cmd.Env`
- Tests call `NewManager`/`NewProcess` directly in ~15 places

## Out of Scope
- Per-process env files
- Overriding env vars via CLI flags
- `.env` auto-discovery (must be explicitly passed with `-e`)

## Approach
- `Env{Key, Value}` struct in `utils.go` alongside `Entry` — same pattern
- `ParseEnvFile(path string) ([]Env, error)` in `utils.go` — returns ordered slice, handles `${VAR}` expansion during parsing
- `NewProcess` accepts `[]Env` (nil means inherit parent env only)
- `Process.Start()` ranges over the slice to build `cmd.Env` inline
- `NewManager` accepts `[]Env` and forwards to each `NewProcess`

## Changes

- [x] **`gopty/env.go`** (new) — Dedicated env file
  - `Env` struct with `Key` and `Value` string fields
  - `ParseEnv(line string) (Env, error)` — parses a single line, trims key/value, returns error for empty/comment/missing separator
  - `Expand(expanded map[string]string) string` — method on Env, expands `${VAR}` references using loop for multiple references per value

- [x] **`gopty/env_test.go`** (new) — Test `ParseEnv` and `Env.Expand`
  - `TestParseEnv`: parses key=value, trims, error for empty/comment/missing separator
  - `TestEnv_Expand`: returns unchanged, expands from map, falls back to os env, handles multiple refs

- [x] **`gopty/utils_test.go`** — Add `TestParseEnvFile` with subtests:
  - Parses key=value pairs
  - Skips comments and blank lines
  - Handles values containing `=`
  - Returns unexpanded values for expansion at process start
  - Returns error for missing file

- [x] **`gopty/process.go`** — Add env support
  - Add `env []Env` field to `Process` struct
  - Change `NewProcess(entry Entry, index int, env []Env) *Process`
  - In `Start()`, if `p.env != nil`, build `expanded` map, call `e.Expand(expanded)` for each env, build `cmd.Env`

- [x] **`gopty/manager.go`** — Thread `[]Env` through
  - Change `NewManager(entries []Entry, stdout io.Writer, env []Env) *Manager`
  - Store env, pass to each `NewProcess` call

- [x] **`cmd/main.go`** — Add `-e` flag
  - Add `envFile := flag.String("e", "", "path to .env file")`
  - If set, call `ParseEnvFile`, fail fast on error
  - Pass `[]Env` (or nil) to `NewManager`

- [x] **`gopty/process_test.go`** — Update `NewProcess` calls to pass `nil` as third arg
- [x] **`gopty/manager_test.go`** — Update `NewManager` calls to pass `nil` as third arg

## Success Criteria

### Automated:
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

### Manual:
- [ ] `.env` with `FOO=bar`, Procfile `web: echo $FOO` → prints `bar`
- [ ] `${VAR}` expansion: `BASE=/app` then `LOG=${BASE}/logs` → `/app/logs`
- [ ] `go-pty -e nonexistent.env` fails with clear error
- [ ] `go-pty` (no `-e`) works as before

## References
- `gopty/env.go` — `Env` struct, `NewEnv`, `Expand` functions
- `gopty/utils.go` — `ParseEnvFile`, `Entry` struct, `ParseProcfile`
- `gopty/process.go:19-27` — `Process` struct
- `gopty/process.go:37-49` — `Start()`
- `gopty/manager.go:20-30` — `NewManager`
