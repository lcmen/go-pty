# Phase 3: Attached mode prefix

## Overview
Add a `[name]` prefix (without color) to attached process output while keeping raw reads for interactivity.

## Current State
- `process.go:189-194` — `OutputAttached` branch does raw `reader.Read(buf)` + `stdout.Write(buf[:n])`, no prefix
- `process_test.go:57-69` — `OutputAttached` test expects raw passthrough

## Out of Scope
- Changing read mode to line-buffered (would break REPLs, prompts, progress bars)

## Approach
Keep raw `reader.Read` in the `OutputAttached` branch. After reading a chunk, scan for `\n` bytes and insert `[name] ` prefix after each one. Also prepend the prefix at the very start of the stream (first chunk). This preserves immediate output while adding per-line prefixes.

## Changes

- [ ] **`gopty/process.go`** — In `read()`, change `OutputAttached` branch
  - Keep raw `reader.Read(buf)`
  - Build a plain prefix string `[name] ` (no color codes)
  - Track whether we're at the start of a line (initially true for first chunk)
  - Scan the chunk: prepend prefix at start-of-line, insert prefix after each `\n`
  - Write the modified chunk to stdout

- [ ] **`gopty/process_test.go`** — Update `OutputAttached` test
  - Expected output changes from raw `"line1\nline2\n"` to `"[web] line1\n[web] line2\n"`

## Success Criteria

### Automated:
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

### Manual:
- [ ] Attach to a process — output shows `[name]` prefix per line without color
- [ ] Interactive processes (irb, rails console) still show prompts immediately
- [ ] Detach — output reverts to colored `[name]` prefix (OutputAll behavior)
