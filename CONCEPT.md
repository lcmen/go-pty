# Terminal & PTY Concepts

A guide to the foundational concepts behind go-pty.

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

**Why go-pty needs raw mode:** The terminal enters raw mode at startup so the Controller can read individual keystrokes (like ctrl+] to open the dialog, or ctrl+c to shutdown). Without raw mode, the TTY driver would buffer input until Enter and intercept Ctrl+C as SIGINT. Raw mode only affects stdin — process output is unaffected and renders normally (though lines must use `\r\n` instead of `\n` since the TTY driver no longer translates newlines).

**Why we must restore:** If go-pty crashes while in raw mode, your terminal stays in raw mode. Your shell is still running, but:
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
|  go-pty   | ◄------► |   TTY Driver    | ◄------► |  child   |
|           |   (fd)   |   (in kernel)   |   (fd)   | process  |
+-----------+          +-----------------+          +----------+
```

- **Master end** — held by go-pty. We read output from it and write keystrokes to it.
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

**Process groups** solve this. `pty.Start()` internally calls `setsid` which creates a new session and process group for each child. On shutdown, go-pty sends SIGINT to the entire process group via `syscall.Kill(-pid, syscall.SIGINT)` (negative pid targets the group). If a process doesn't exit within 5 seconds, it escalates to SIGKILL.

---

## Signals

Signals are the kernel's way of poking a process. The relevant ones for go-pty:

| Signal | Default action | Sent when |
|--------|---------------|-----------|
| SIGINT | Terminate | User presses Ctrl+C (in cooked mode) |
| SIGTERM | Terminate | Polite "please exit" from another process |
| SIGKILL | Terminate (cannot be caught) | Force kill — last resort |
| SIGWINCH | Ignored | Terminal window is resized |
| SIGTSTP | Suspend | User presses Ctrl+Z (in cooked mode) |

go-pty listens for:
- **SIGINT / SIGTERM** — triggers graceful shutdown (SIGINT to all children, wait up to 5s, SIGKILL stragglers, restore terminal, exit)
- **SIGWINCH** — propagates the new terminal size to all child PTYs so they reflow their output

---

## Terminal Size and SIGWINCH

Terminals have a width and height (e.g., 80x24). Programs query this to format output (wrapping text, drawing progress bars, table layouts).

When you resize your terminal window:
1. The kernel sends **SIGWINCH** to the foreground process (go-pty)
2. go-pty reads the new size via `pty.GetsizeFull(os.Stdin)` and calls `pty.Setsize(p.master, ws)` for each child
3. This copies the new dimensions from our real terminal to each PTY
4. The kernel sends SIGWINCH to each child process
5. Children re-query their terminal size and reflow output

Without this propagation, child processes would think the terminal is still the original size and format output incorrectly after a resize.

---

## `sh -c "<command>"` — Why the Wrapper?

Procfile commands can be complex:

```
web: PORT=3000 bundle exec rails server -b 0.0.0.0
worker: bundle exec sidekiq -c 5 && echo "done"
```

These use shell features (env vars, `&&`). We can't pass them directly to `exec.Command` — that takes a binary path and arguments, not shell syntax. So we use `sh -c` to let the shell parse the command.

Since `pty.Start()` calls `setsid` to create a new session and process group, `kill(-pid)` targets the entire process group — both the `sh` wrapper and all its children.

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

## Atomic Operations — Lock-Free Attach/Detach

go-pty runs a goroutine per process, all reading output concurrently. Each goroutine checks its process's output mode on every read iteration to decide how to route output (prefixed, raw, or ignored).

Since this is a hot path (called per chunk of output), each process stores its mode in an `atomic.Value` instead of behind a mutex:

```go
type Process struct {
    mode atomic.Value  // stores OutputMode
    // ...
}
```

`atomic.Value` provides lock-free reads and writes — goroutines never block each other. The `read` loop on each process loads the mode atomically to decide how to handle the current chunk:

```go
mode := p.mode.Load().(OutputMode)
switch mode {
case OutputAll:      // prefix each line and write
case OutputAttached: // write raw bytes directly
case OutputIgnored:  // read and discard
}
```

When the manager attaches or detaches a process, it updates all processes' modes at once via `updateAllModes` — the attached process gets `OutputAttached`, all others get `OutputIgnored`, and on detach everyone returns to `OutputAll`.

Note: concurrent writes to stdout from multiple goroutines can still interleave. In practice, `fmt.Fprintf` with short lines rarely produces garbled output, and the colored prefixes make it clear which process each line belongs to.

---

## Line Scanning

PTY reads return arbitrary chunks of data, not neat lines. go-pty wraps the PTY master in a `bufio.Reader` and uses `ReadBytes('\n')` to yield complete lines in `OutputAll` mode. In `OutputAttached` and `OutputIgnored` modes, it reads raw chunks with `Read()` instead — attached output needs to be forwarded immediately (no line buffering), and ignored output just needs to be drained:

```go
switch p.outputMode() {
case OutputAll:
    line, err = p.reader.ReadBytes('\n')
    // prefix and write line
case OutputAttached:
    n, err = p.reader.Read(buf)
    // write raw bytes
case OutputIgnored:
    n, err = p.reader.Read(buf)
    // discard
}
```

Since the terminal is in raw mode, output uses `\r\n` (carriage return + newline) instead of `\n`. Without the `\r`, each line would start further to the right because raw mode disables the TTY driver's automatic newline-to-carriage-return translation.

## Alternate Screen Buffer

Terminals support two screen buffers: the **normal screen** (where your shell output lives) and the **alternate screen** (used by full-screen programs like vim, less, htop).

go-pty uses the alternate screen for the process selection dialog:

```
\033[?1049h  — enter alternate screen (saves cursor, clears screen)
\033[?1049l  — leave alternate screen (restores previous content)
```

When the dialog opens, it switches to the alternate screen, draws the process list, and handles arrow key navigation. When the user selects a process or presses Esc, it switches back — the original output is restored exactly as it was. Any process output that arrived during the dialog is harmless (it writes to the alternate screen and gets discarded on switch back).
