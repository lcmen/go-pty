# Phase 1: Filter flag (`-s`)

## Overview
Add a `-s` flag that accepts a comma-separated list of process names (e.g. `-s web,worker`) to run only a subset of processes from the Procfile.

## Current State
- `cmd/main.go:30` — `-f` flag exists for Procfile path
- `gopty/utils.go:58` — `ParseProcfile` returns `[]Entry`
- `cmd/main.go:57-69` — `initManager` passes all entries directly to `NewManager`
- No filtering capability exists

## Out of Scope
- Pattern/glob matching for names (exact match only)
- Per-process env files or config

## Approach
Add `FilterEntries` to `utils.go` as a pure function that validates and filters. Call it in `main.go` between `ParseProcfile` and `NewManager`. Fail fast if any requested name doesn't exist.

## Changes

- [ ] **`gopty/utils.go`** — Add `FilterEntries(entries []Entry, names []string) ([]Entry, error)`
  - Return only entries whose `Name` matches one of the given names
  - If a name in `names` doesn't match any entry, return error listing the unknown name(s)
  - If `names` is empty, return all entries unchanged (no-op)

- [ ] **`gopty/utils_test.go`** — Add `TestFilterEntries` with subtests:
  - Filters entries to matching names
  - Returns all entries when names is empty/nil
  - Returns error for unknown names

- [ ] **`cmd/main.go`** — Add `-s` flag and wire filtering
  - Add `serviceFilter := flag.String("s", "", "comma-separated list of services to run")`
  - In `initManager`, accept `filter string` parameter
  - If filter is non-empty, split on `,`, trim whitespace, call `FilterEntries` before `NewManager`
  - Update `flag.Usage` to document the new flag

## Success Criteria

### Automated:
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

### Manual:
- [ ] `go-pty -s web` runs only the `web` process
- [ ] `go-pty -s web,worker` runs both
- [ ] `go-pty -s nonexistent` fails with a clear error
- [ ] `go-pty` (no flag) runs all processes as before

## References
- `cmd/main.go:30` — existing `-f` flag pattern
- `gopty/utils.go:53-56` — `Entry` struct
- `gopty/utils.go:58-93` — `ParseProcfile`
- `gopty/utils_test.go:12-48` — existing test patterns
