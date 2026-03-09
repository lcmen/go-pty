# AGENTS.md

Guidelines and reference for AI agents working on the go-pty project.

## Project Overview

**go-pty** is a process manager that runs commands from a Procfile, each in its own pseudoterminal (PTY). It enables interactive debugging by allowing users to attach to any running process and use REPLs/debuggers like `binding.irb`, `byebug`, etc.

- **Repository**: `git@github.com:lcmen/go-pty.git`
- **Language**: Go 1.26+
- **License**: MIT

## Architecture

### Components

| File | Responsibility |
|------|---------------|
| `cmd/main.go` | Entry point, CLI flags, signal handling, raw terminal mode |
| `gopty/manager.go` | Manages multiple processes, handles attach/detach logic |
| `gopty/controller.go` | Handles keyboard input, mode switching, dialog invocation |
| `gopty/process.go` | Individual PTY process lifecycle, output streaming |
| `gopty/dialog.go` | Interactive process selection UI using alternate screen |
| `gopty/utils.go` | Procfile parsing, utilities, constants |

### Data Flow

```
User Input → Controller → Manager → Process (PTY Master) → Child Process (PTY Slave)
                                           ↓
                                    Output Stream → Stdout (with prefixes)
```

### Output Modes

Processes operate in three modes controlled via `atomic.Value`:

- `OutputAll` - Prefix each line and display to stdout
- `OutputAttached` - Forward raw bytes directly to stdout (bypass prefixing)
- `OutputIgnored` - Read and discard output (prevents blocking)

## Key Concepts

### Pseudoterminals (PTY)

A PTY is a kernel-managed fake terminal with two ends:
- **Master** - Held by go-pty, we read/write here
- **Slave** - Given to child process as stdin/stdout/stderr

Why PTY over pipes? Programs detect terminals and change behavior (colors, progress bars, interactive features). A PTY makes children believe they're connected to a real terminal.

### Raw Terminal Mode

The terminal runs in raw mode so the Controller can capture individual keystrokes:
- `ctrl+]` opens the attach dialog
- `ctrl+c` triggers shutdown
- Arrow keys navigate the dialog

**Critical**: Always `defer` terminal restore. If the program crashes in raw mode, the terminal becomes unusable (no echo, no line editing). Users would need to blindly type `reset` + Enter.

### Process Groups

Each child runs in its own process group via `setsid`. This allows sending signals to the entire tree:

```go
syscall.Kill(-pid, syscall.SIGINT)  // Negative PID = process group
```

### Signals

- `SIGINT` / `SIGTERM` - Graceful shutdown (SIGINT to children, wait 5s, then SIGKILL)
- `SIGWINCH` - Terminal resize, propagated to all child PTYs

## Code Patterns

### Atomic Operations

Use `atomic.Value` for lock-free mode switching (hot path called per output chunk):

```go
type Process struct {
    mode atomic.Value  // stores OutputMode
}

mode := p.mode.Load().(OutputMode)
```

### PTY Locking

PTY operations use RWMutex with helper functions:

```go
unlock, err := p.lock(PtyReadLock)
if err != nil {
    return err
}
defer unlock()
```

### Error Handling

- Wrap errors with context: `fmt.Errorf("failed to start process %s: %w", name, err)`
- Controller stores errors in `atomic.Pointer[error]` for goroutine-safe access

## Development Commands

```bash
# Build
make build          # Build binary to ./go-pty

# Test
make test           # Run tests with race detection

# Lint
make lint           # Run go vet and go fmt

# Stats
make stats          # Show code statistics

# Install
make install        # Build and copy to ~/.local/bin
```

## Testing Guidelines

### Test Structure

- Use subtests via `t.Run()` for organizing test cases
- Group related tests with `t.Run()` subtests
- Tests are in `gopty/*_test.go` files

### Test Helpers

```go
// Create a stub process with mocked PTY for Stream tests
func stubProcess(t *testing.T, input string, exitCode int) (*Process, *os.File)

// Create a test process with a real command for lifecycle tests
func stubProcessWithCommand(t *testing.T, command string) *Process
```

### Test Utilities

- Use `syncBuffer` for goroutine-safe output capture in tests
- Use `google/go-cmp` for assertions: `cmp.Diff(expected, got)`
- Use `waitFor()` helper for async conditions with timeout

### Common Patterns

```go
t.Run("description", func(t *testing.T) {
    var buf syncBuffer
    m := NewManager(entries, &buf)
    
    if err := m.StartAll(); err != nil {
        t.Fatalf("StartAll failed: %v", err)
    }
    
    waitFor(t, func() bool { 
        return strings.Contains(buf.String(), "expected") 
    })
})
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/creack/pty` | PTY creation, resizing, size queries |
| `golang.org/x/term` | Terminal raw mode (`MakeRaw`, `Restore`) |
| `golang.org/x/sys` | System calls (signals, syscalls) |
| `github.com/google/go-cmp` | Test assertions (dev dependency) |

## Procfile Format

```
# Comment
web: bundle exec rails server -p 3000
worker: bundle exec sidekiq
css: tailwindcss --watch
```

- Format: `name: command`
- Empty lines and `#` comments are ignored
- Commands can include shell syntax (env vars, `&&`, etc.) via `sh -c` wrapper

## Keyboard Shortcuts

| Mode | Key | Action |
|------|-----|--------|
| Normal | `ctrl+]` | Open process selection dialog |
| Normal | `ctrl+c` | Shut down all processes and exit |
| Dialog | `↑/↓` | Navigate process list |
| Dialog | `Enter` | Attach to selected process |
| Dialog | `Esc` | Cancel |
| Attached | `ctrl+]` | Detach and return to normal mode |

## Important Implementation Details

### Line Endings

In raw mode, the TTY driver doesn't translate newlines. Output must use `\r\n` (carriage return + newline) instead of just `\n`.

### Alternate Screen Buffer

The dialog uses the terminal's alternate screen buffer:
- Enter: `\033[?1049h`
- Leave: `\033[?1049l`

This preserves the original output when the dialog closes.

### Shutdown Sequence

1. Send `SIGINT` to all process groups
2. Wait up to 5 seconds for graceful exit
3. Send `SIGKILL` to any remaining processes
4. Restore terminal state
5. Exit

## CI/CD

- GitHub Actions workflow in `.github/workflows/ci.yml`
- Runs on: push to main, pull requests
- Jobs: test (with race detector), build

## Common Tasks

### Adding a New Keyboard Shortcut

1. Add constant in `utils.go` (e.g., `byteCtrlD = 4`)
2. Handle in `controller.go` `handleAllOut()` or `handleAttached()`
3. Add test in `controller_test.go`

### Modifying Output Behavior

1. Update `OutputMode` enum if needed
2. Modify `Process.read()` in `process.go`
3. Update `Manager.updateAllModes()` if mode transitions change
4. Add/update tests in `process_test.go` and `manager_test.go`

### Adding New PTY Operations

1. Use `p.lock(PtyReadLock)` or `p.lock(PtyWriteLock)` helper
2. Handle nil PTY case (returns error)
3. Defer unlock in all paths
4. Add tests with stub processes

## References

- `CONCEPT.md` - Detailed explanation of PTY, TTY, signals, and terminal concepts
- `README.md` - User-facing documentation
- `CHANGELOG.md` - Version history
