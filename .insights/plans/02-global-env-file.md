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

- [ ] **`gopty/utils.go`** — Add `Env` struct and `ParseEnvFile`
  - `Env` struct with `Key` and `Value` string fields
  - `ParseEnvFile(path string) ([]Env, error)` — skip blank lines and `#` comments, split on first `=`, expand `${VAR}` references using values parsed so far

- [ ] **`gopty/utils_test.go`** — Add `TestParseEnvFile` with subtests:
  - Parses key=value pairs
  - Skips comments and blank lines
  - Handles values containing `=`
  - Expands `${VAR}` references
  - Returns error for missing file

- [ ] **`gopty/process.go`** — Add env support
  - Add `env []Env` field to `Process` struct
  - Change `NewProcess(entry Entry, index int, env []Env) *Process`
  - In `Start()`, if `p.env != nil`, range over slice and build `cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", e.Key, e.Value)...)` inline

- [ ] **`gopty/manager.go`** — Thread `[]Env` through
  - Change `NewManager(entries []Entry, stdout io.Writer, env []Env) *Manager`
  - Store env, pass to each `NewProcess` call

- [ ] **`cmd/main.go`** — Add `-e` flag
  - Add `envFile := flag.String("e", "", "path to .env file")`
  - If set, call `ParseEnvFile`, fail fast on error
  - Pass `[]Env` (or nil) to `NewManager`

- [ ] **`gopty/process_test.go`** — Update `NewProcess` calls to pass `nil` as third arg
- [ ] **`gopty/manager_test.go`** — Update `NewManager` calls to pass `nil` as third arg

## Success Criteria

### Automated:
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

### Manual:
- [ ] `.env` with `FOO=bar`, Procfile `web: echo $FOO` → prints `bar`
- [ ] `${VAR}` expansion: `BASE=/app` then `LOG=${BASE}/logs` → `/app/logs`
- [ ] `go-pty -e nonexistent.env` fails with clear error
- [ ] `go-pty` (no `-e`) works as before

## References
- `gopty/utils.go:53-56` — `Entry` struct (same pattern for `Env`)
- `gopty/utils.go:58-93` — `ParseProcfile` (same pattern for `ParseEnvFile`)
- `gopty/process.go:19-27` — `Process` struct
- `gopty/process.go:37-49` — `Start()`
- `gopty/manager.go:20-30` — `NewManager`
