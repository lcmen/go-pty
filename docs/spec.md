# PTY Multiplexer (pty-mux) — Implementation Specification

## Overview

A Go command-line tool that runs multiple commands concurrently, each in its own PTY (pseudoterminal). By default it shows prefixed output from all commands. The user can "step into" a single process for full interactive terminal access (e.g. `binding.pry`, `byebug`, `pdb`).

## Usage

```bash
pty-mux              # reads ./Procfile
pty-mux Procfile.dev # reads a specific Procfile
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

## Architecture

### Core Concepts

- Each command from the Procfile is spawned in its own PTY using `pty.Start()`
- The child process gets the PTY slave end as its fd 0/1/2 — it thinks it's a real terminal
- The Go program holds the PTY master end — reads output and writes keystrokes through it
- Commands are run via `sh -c "exec <command>"` so shell features work (pipes, &&, env vars) and `exec` eliminates the extra `sh` wrapper process
- Each child gets its own process group (`Setpgid: true`) so we can kill the entire process tree with `syscall.Kill(-pid, SIGTERM)`

### Two Modes

**All Output Mode (default)**
- A goroutine per process reads from each PTY master
- Output is buffered per process until a complete line (`\n`) is received
- Complete lines are printed with a colored prefix: `[web] Starting server on port 3000`
- A mutex protects stdout writes so lines from different processes don't interleave
- User types `!1` or `!web` + Enter to step into a process

**Stepped In Mode**
- The real terminal is switched to raw mode (`term.MakeRaw`)
- Every keystroke from stdin is forwarded directly to the attached process's PTY master
- The attached process's output goes directly to stdout (no prefix, no buffering)
- Other processes' output is buffered (not printed), with a 1MB cap per process — oldest lines are dropped when the cap is exceeded, with a count of dropped lines tracked
- `ctrl+]` (byte value 29) detaches and returns to all output mode
- On detach, buffered output from other processes is flushed with prefixes, showing drop counts if any

### Data Structures

```go
type Process struct {
    label   string       // name from Procfile
    args    []string     // command args
    cmd     *exec.Cmd
    master  *os.File     // PTY master fd
    color   string       // ANSI color code for prefix
    lineBuf string       // accumulates partial lines between reads
    dropped int          // lines dropped due to buffer overflow
}

type Manager struct {
    processes []*Process
    attached  *Process       // nil = all output mode
    rawState  *term.State    // saved terminal state before raw mode
    mu        sync.Mutex     // protects attached, lineBuf, dropped, stdout
    wg        sync.WaitGroup // tracks readOutput goroutines for clean shutdown
}
```

### Concurrency & Mutex Rules

All reads and writes to `m.attached`, `p.lineBuf`, and `p.dropped` must hold `m.mu`. The mutex also protects stdout writes to prevent interleaving. The mutex is NOT held during blocking `Read()` calls — only around the processing/printing logic after a read completes.

Pattern in readOutput:
```
1. Read from PTY master (blocking, no mutex)
2. Lock mutex
3. Append to lineBuf
4. Check mode (attached == this process? different process? nil?)
5. Print or buffer accordingly
6. Unlock mutex
```

### Line Buffering Logic

PTY reads return arbitrary chunks, not complete lines. Each process has a `lineBuf` that accumulates data across reads:

1. Every read appends to `lineBuf`
2. In all output mode: extract and print every complete line (up to `\n`), keep the remainder
3. In stepped-in mode for this process: write raw to stdout, clear buffer
4. In stepped-in mode for another process: just buffer, trim to 1MB if needed by dropping oldest lines at line boundaries

When a process exits, flush any remaining partial line in the buffer.

### Buffer Overflow Handling

When stepped into a different process and a buffer exceeds 1MB:
1. Calculate how many lines are in the excess portion
2. Add that count to `p.dropped`
3. Trim from the front at a line boundary (find next `\n` after cut point)
4. On step out, print drop count before flushing: `[worker] ... 847 lines dropped (buffer full) ...`

### Process Spawning

```go
cmd := exec.Command("sh", "-c", "exec " + command)
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
master, err := pty.Start(cmd)
```

- `sh -c` handles shell parsing (quotes, pipes, variables)
- `exec` prefix replaces `sh` with the actual command (no extra process)
- `Setpgid: true` creates a new process group so `Kill(-pid)` reaches all descendants

### Terminal Resize (SIGWINCH)

Listen for SIGWINCH signal. On resize, call `pty.InheritSize(os.Stdin, p.master)` for all processes to propagate the new terminal dimensions to each PTY. Also set initial size on startup.

### Graceful Shutdown

On SIGINT (ctrl+c in all output mode) or SIGTERM:
1. Restore terminal from raw mode if needed
2. Send SIGTERM to each process group: `syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)`
3. Wait for all readOutput goroutines to finish (`wg.Wait()`)
4. Exit

### Terminal Safety

- `defer` at top of main restores terminal state on panic
- `rawState` is set to `nil` after restore to prevent double-restore
- Shutdown handler restores terminal before printing

### Step In/Out Flow

**Step In:**
1. Print instructions ("ctrl+] to detach")
2. Save terminal state with `term.MakeRaw(os.Stdin.Fd())`
3. Set `m.attached = p` (under mutex)
4. handleStdin loop now forwards all bytes to `p.master.Write()` except ctrl+]

**Step Out:**
1. Restore terminal with `term.Restore()`, set rawState to nil
2. Set `m.attached = nil` (under mutex)
3. Flush buffered output from all processes (with drop counts)
4. Print instructions for all output mode

### Process Exit

When `p.master.Read()` returns an error:
1. Lock mutex
2. Flush remaining partial line in lineBuf
3. Print exit code from `p.cmd.ProcessState.ExitCode()`
4. Unlock mutex
5. Decrement WaitGroup

### Color Assignment

Processes are assigned colors in order from a palette:
```
red, green, yellow, blue, magenta, cyan
```
Colors wrap around with modulo for more than 6 processes.

### User Commands (All Output Mode)

- `!N` + Enter — step into process N (1-indexed)
- `!name` + Enter — step into process by Procfile label (case-insensitive)
- ctrl+c — kill all processes and exit

### User Commands (Stepped In Mode)

- ctrl+] — detach, return to all output mode
- All other keystrokes forwarded to the attached process
