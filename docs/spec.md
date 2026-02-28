# go-pty — Implementation Specification

## Overview

A Go command-line tool that runs multiple commands concurrently, each in its own PTY (pseudoterminal). By default it shows prefixed output from all commands. The user can attach to a single process via a dialog modal for full interactive terminal access (e.g. `binding.pry`, `byebug`, `pdb`).

## Usage

```bash
go-pty                    # reads ./Procfile
go-pty -f Procfile.dev    # reads a specific Procfile
```

## Procfile Format

Standard Heroku/foreman format:

```
# Comments start with #
web: bundle exec rails server -p 3000
worker: bundle exec sidekiq
css: tailwindcss --watch
```

Each line is `name: command`. Empty lines and `#` comments are skipped.

## Dependencies

```
github.com/creack/pty   — spawn processes in a PTY
golang.org/x/term       — raw mode, terminal state save/restore
```

## Architecture (MVC)

### Core Concepts

- Each command from the Procfile is spawned in its own PTY using `pty.Start()`
- The child process gets the PTY slave end as its fd 0/1/2 — it thinks it's a real terminal
- The Go program holds the PTY master end — reads output and writes keystrokes through it
- Commands are run via `sh -c "exec <command>"` so shell features work (pipes, &&, env vars) and `exec` eliminates the extra `sh` wrapper process
- `pty.Start()` internally calls `setsid` to create a new session for each child
- Terminal enters raw mode at startup so the Controller can read individual keystrokes
- Output uses `\r\n` for raw mode compatibility

### Three States

**Normal (All Output) — default**
- A goroutine per process reads from each PTY master via `bufio.Scanner`
- Complete lines are printed with a colored prefix: `[web] Starting server on port 3000`
- ctrl+] opens the process selection dialog
- ctrl+c shuts down go-pty

**Dialog (Alternate Screen)**
- Switches to the alternate screen buffer
- Displays a numbered process list with colors and arrow key highlight
- Arrow keys navigate, Enter selects (attaches), Esc cancels (returns to normal)
- Dialog.Open() is blocking — owns its own input loop, returns `(index, bool)`

**Attached**
- The attached process's output goes directly to stdout (no prefix)
- Other processes' output is dropped (not buffered)
- ctrl+] detaches and returns to normal mode
- Other keystrokes are forwarded to the attached process's PTY master (Phase 5b)

### Data Structures

```go
type Process struct {
    Color      string          // ANSI color code for prefix
    Entry                      // Name + Command from Procfile
    cmd        *exec.Cmd
    master     *os.File        // PTY master fd (bidirectional)
    outputMode func() OutputMode // callback to determine routing
}

type Manager struct {
    attached  atomic.Pointer[Process] // nil = normal mode, lock-free
    out       io.Writer
    processes []*Process
    wg        sync.WaitGroup          // tracks read goroutines
}

type Controller struct {
    err     error       // set to io.EOF to stop Run loop
    manager *Manager
    stdin   io.Reader
    stdout  io.Writer
}

type Dialog struct {
    in        io.Reader
    out       io.Writer
    processes []*Process
    selected  int
}
```

### Concurrency

`atomic.Pointer[Process]` provides lock-free reads on the hot path (called per line of output in each goroutine). No mutex is needed — the `outputMode` callback captures the manager and process, performing an atomic load to check attachment status.

### Output Routing

Each process has an `outputMode` callback assigned at construction by `Manager.outputMode(p)`. The callback returns one of three modes:

- `OutputAll` — no process attached, print with colored prefix
- `OutputAttached` — this process is attached, print raw
- `OutputIgnored` — another process is attached, drop output

### Process Spawning

```go
cmd := exec.Command("sh", "-c", "exec " + command)
master, err := pty.Start(cmd)
```

- `sh -c` handles shell parsing (quotes, pipes, variables)
- `exec` prefix replaces `sh` with the actual command (no extra process)
- `pty.Start()` calls `setsid` internally to create a new session

### Terminal Resize (SIGWINCH)

Listen for SIGWINCH signal. On resize, read new size via `pty.GetsizeFull(os.Stdin)` and call `pty.Setsize(p.master, ws)` for all processes to propagate the new terminal dimensions to each PTY. Also set initial size on startup in `StartAll()`.

### Graceful Shutdown

On SIGINT/SIGTERM or ctrl+c in normal mode:
1. `Controller.Shutdown()` sets `c.err = io.EOF` and calls `m.Shutdown()`
2. `Manager.Shutdown()` sends SIGTERM to each process via `p.cmd.Process.Signal(syscall.SIGTERM)`
3. Waits for all read goroutines to finish (`wg.Wait()`)
4. `defer restore()` in main restores terminal from raw mode

### Terminal Safety

- `rawMode(os.Stdin)` returns a restore closure, called via `defer` at top of main
- Even on panic, the deferred restore runs and the terminal is returned to normal

### Attach/Detach Flow

**Attach (via Dialog):**
1. ctrl+] opens Dialog on alternate screen
2. User navigates with arrow keys, selects with Enter
3. `Dialog.Open()` returns selected index
4. Controller calls `Manager.Attach(index)` — atomic store
5. Output routing changes immediately via `outputMode` callbacks

**Detach:**
1. ctrl+] in attached mode
2. Controller calls `Manager.Detach()` — atomic store of nil
3. Output routing reverts to prefixed mode

### Process Exit

When `bufio.Scanner.Scan()` returns false (PTY master closed):
1. Call `p.cmd.Wait()` to get exit status
2. Print exit message: `[name] exited (code N)`
3. Goroutine returns, decrementing WaitGroup

### Color Assignment

Processes are assigned colors in order from a 12-color palette:
```
red, green, yellow, blue, magenta, cyan,
bright red, bright green, bright yellow, bright blue, bright magenta, bright cyan
```
Colors wrap around with modulo for more than 12 processes.

### User Commands

**Normal mode:**
- ctrl+] — open process selection dialog
- ctrl+c — shut down go-pty (SIGTERM all processes)

**Dialog mode:**
- Arrow up/down — navigate process list
- Enter — select and attach to highlighted process
- Esc — cancel, return to normal mode

**Attached mode:**
- ctrl+] — detach, return to normal mode
- All other keystrokes forwarded to the attached process (Phase 5b)
