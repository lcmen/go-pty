# Terminal & PTY Concepts

A guide to the foundational concepts behind pty-mux.

---

## What is a Terminal?

Historically, a terminal was a physical device — a screen and keyboard connected to a mainframe. Today, when you open "Terminal.app" or "iTerm2" or "Windows Terminal", you're running a **terminal emulator**: software pretending to be that old hardware.

Your terminal emulator does two things:
1. Draws text on screen
2. Sends your keystrokes to a program (usually your shell — bash, zsh, fish)

Between the terminal emulator and your shell sits a kernel component called the **TTY driver**. This is where a lot of the "magic" happens.

---

## The TTY Driver — Your Invisible Editor

When you type in a normal terminal, you're not talking directly to your shell. The kernel's TTY driver sits in the middle and processes your input before the shell ever sees it.

```
You type keys → TTY Driver → Shell receives lines
```

The TTY driver provides **line editing** in what's called **cooked mode** (the default):

| You press | What happens |
|-----------|-------------|
| Regular keys | Characters are echoed to screen AND buffered |
| Backspace | Deletes last character from the buffer and screen |
| Enter | Sends the complete line to the program |
| Ctrl+C | Sends SIGINT signal to the foreground process (kills it) |
| Ctrl+D | Sends EOF (end of input) |
| Ctrl+Z | Sends SIGTSTP (suspends process to background) |

Key point: **in cooked mode, the program only receives complete lines after you press Enter**. The TTY driver handles all the character-by-character editing.

---

## Raw Mode — Why We Need It

Raw mode tells the TTY driver: "stop helping, give me everything."

| Feature | Cooked Mode (default) | Raw Mode |
|---------|----------------------|----------|
| Echoing keystrokes | TTY driver echoes automatically | Nothing echoed — program must do it |
| Backspace | Deletes from buffer | Sent as a byte (value 127) to the program |
| Ctrl+C | Kernel sends SIGINT | Sent as a byte (value 3) to the program |
| Line buffering | Program gets complete lines | Program gets every keystroke immediately |
| Ctrl+Z, Ctrl+D | Kernel handles them | Just bytes to the program |

**Why pty-mux needs raw mode:** When you step into a process, we need to forward *every* keystroke to that process immediately. If the TTY driver was buffering lines and intercepting Ctrl+C, the attached process would never receive those keystrokes. An interactive debugger needs to receive raw input to work properly.

**Why we must restore:** If pty-mux crashes while in raw mode, your terminal stays in raw mode. Your shell is still running, but:
- Nothing you type is echoed on screen
- Backspace doesn't work visually
- Ctrl+C doesn't kill anything (it's just byte 3 going to your shell)
- No line editing at all

You'd have to blindly type `reset` + Enter to fix it. That's why we `defer` a terminal restore at the top of main — even if the program panics, the terminal gets restored.

---

## What is a PTY (Pseudoterminal)?

A PTY is a fake terminal created by the kernel. It has two ends:

```
+-----------+          +------ PTY ------+
|           |  master  |                 |  slave   +----------+
|  pty-mux  | ◄------► |   TTY Driver    | ◄------► |  child   |
|           |   (fd)   |   (in kernel)   |   (fd)   | process  |
+-----------+          +-----------------+          +----------+
```

- **Master end** — held by pty-mux. We read output from it and write keystrokes to it.
- **Slave end** — given to the child process as its stdin/stdout/stderr (fd 0, 1, 2). The child has no idea it's not a real terminal.

**Why not just use pipes?** Many programs change behavior when they detect they're not connected to a terminal:
- `ls` drops colors and columns
- `grep` drops colors
- Progress bars and spinners disappear
- Interactive tools (debuggers, REPLs) may refuse to run
- Programs buffer output differently (line-buffered for terminals, block-buffered for pipes)

A PTY makes the child process believe it's connected to a real terminal, so all these behaviors work normally.

### How a PTY is created

```go
cmd := exec.Command("sh", "-c", "exec bundle exec rails server")
master, err := pty.Start(cmd)  // creates PTY, starts process with slave end
```

`pty.Start` does several things under the hood:
1. Calls `posix_openpt()` — creates a new PTY pair (master + slave)
2. Sets the slave end as the child's stdin/stdout/stderr
3. Starts the child process
4. Returns the master end to us

---

## Process Groups — Killing the Whole Tree

When you run `bundle exec rails server`, the process tree might look like:

```
sh (pid 100, pgid 100)
  └── rails (pid 101, pgid 100)
       ├── puma worker 1 (pid 102, pgid 100)
       └── puma worker 2 (pid 103, pgid 100)
```

If we only `kill(101)`, the puma workers become orphans and keep running. That's a resource leak.

**Process groups** solve this. By setting `Setpgid: true`, each command gets its own process group:

```go
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
```

Now `kill(-pid, SIGTERM)` (note the negative pid) sends the signal to every process in that group. The entire tree gets terminated.

---

## Signals

Signals are the kernel's way of poking a process. The relevant ones for pty-mux:

| Signal | Default action | Sent when |
|--------|---------------|-----------|
| SIGINT | Terminate | User presses Ctrl+C (in cooked mode) |
| SIGTERM | Terminate | Polite "please exit" from another process |
| SIGWINCH | Ignored | Terminal window is resized |
| SIGTSTP | Suspend | User presses Ctrl+Z (in cooked mode) |

pty-mux listens for:
- **SIGINT / SIGTERM** — triggers graceful shutdown (kill all children, restore terminal, exit)
- **SIGWINCH** — propagates the new terminal size to all child PTYs so they reflow their output

---

## Terminal Size and SIGWINCH

Terminals have a width and height (e.g., 80x24). Programs query this to format output (wrapping text, drawing progress bars, table layouts).

When you resize your terminal window:
1. The kernel sends **SIGWINCH** to the foreground process (pty-mux)
2. pty-mux calls `pty.InheritSize(os.Stdin, p.master)` for each child
3. This copies the new dimensions from our real terminal to each PTY
4. The kernel sends SIGWINCH to each child process
5. Children re-query their terminal size and reflow output

Without this propagation, child processes would think the terminal is still the original size and format output incorrectly after a resize.

---

## `sh -c "exec <command>"` — Why the Wrapper?

Procfile commands can be complex:

```
web: PORT=3000 bundle exec rails server -b 0.0.0.0
worker: bundle exec sidekiq -c 5 && echo "done"
```

These use shell features (env vars, `&&`). We can't pass them directly to `exec.Command` — that takes a binary path and arguments, not shell syntax. So we use `sh -c` to let the shell parse the command.

But `sh -c "bundle exec rails"` creates an extra process:

```
sh (pid 100)         ← unnecessary wrapper
  └── rails (pid 101)
```

The `exec` prefix replaces the shell with the actual command:

```
sh -c "exec bundle exec rails"

# sh starts, then exec replaces it:
rails (pid 100)      ← sh is gone, rails took its place
```

This matters because:
- `kill(-pid)` targets the right process group leader
- No zombie `sh` processes hanging around
- One fewer process per Procfile entry

---

## File Descriptors 0, 1, 2

Every Unix process starts with three open file descriptors:

| fd | Name | Purpose |
|----|------|---------|
| 0 | stdin | Input (keyboard by default) |
| 1 | stdout | Normal output |
| 2 | stderr | Error output |

When we create a PTY and give the slave end to a child process, the slave fd is duplicated onto 0, 1, and 2. The child reads input from the PTY slave (fd 0) and writes output to it (fd 1, 2). We see that output by reading from the master end.

---

## Mutexes — Preventing Garbled Output

pty-mux runs a goroutine per process, all reading output concurrently. Without synchronization, two goroutines writing to stdout simultaneously could produce garbled output:

```
[web] Starting se[worker] Sidekiq 7.0 starting
rver on port 3000
```

A **mutex** (mutual exclusion lock) ensures only one goroutine writes to stdout at a time:

```go
m.mu.Lock()
fmt.Printf("[%s] %s\n", p.label, line)
m.mu.Unlock()
```

The mutex also protects shared state like `m.attached` (which process we're stepped into) and `p.lineBuf` (partial output buffers). The critical rule: **never hold the mutex during a blocking operation** (like reading from a PTY), or you'll freeze the whole program.

---

## Line Buffering

PTY reads return arbitrary chunks of data, not neat lines:

```
Read 1: "Starting ser"
Read 2: "ver on port 3000\nReady\nAcce"
Read 3: "pting connections\n"
```

We need to reassemble these into complete lines before printing with prefixes. Each process has a `lineBuf` string that accumulates data across reads:

1. Append new data to `lineBuf`
2. Find all complete lines (ending with `\n`)
3. Print each with the colored prefix
4. Keep the remainder (no `\n` yet) in `lineBuf`

When a process exits, we flush whatever's left in `lineBuf` as a final partial line.
