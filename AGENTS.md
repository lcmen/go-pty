# Agent Guidelines for go-pty

## Project Documentation

- [docs/spec.md](docs/spec.md) — Full implementation specification (architecture, data structures, concurrency rules)
- [docs/plan.md](docs/plan.md) — Development plan with phased TDD checklist
- [docs/concepts.md](docs/concepts.md) — Background concepts (PTYs, raw mode, process groups, signals)

## Development Process

This project uses **test-driven development (TDD)**. Each phase follows red-green-refactor:

1. Write failing tests
2. Implement just enough code to pass them
3. Refactor

## Important: Manual Review Required

**Every phase requires manual review before moving to the next.** Do not automatically proceed to the next phase after completing one. Stop after each phase and wait for explicit approval to continue.

The phases are defined in [docs/plan.md](docs/plan.md). Update the status checkboxes in that file as work progresses.

## Project Layout

```
cmd/main.go       ← package main (thin CLI entry point)
gopty/            ← package gopty (all core logic and tests)
```

- `cmd/main.go` handles only CLI args, signal wiring, and calling into `gopty`
- `gopty/` contains all logic and tests
- Tests live alongside source files (`*_test.go`) inside `gopty/`

## Conventions

- Dependencies: `github.com/creack/pty`, `golang.org/x/term`
- Commands are spawned via `sh -c "exec <command>"` — env vars are expanded by the shell
- Build: `go build -o go-pty ./cmd`
- Test: `go test ./gopty/`
