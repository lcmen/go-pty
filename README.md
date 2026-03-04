```
  __ _  ___        _ __ | |_ _   _
 / _` |/ _ \ _____| '_ \| __| | | |
| (_| | (_) |_____| |_) | |_| |_| |
 \__, |\___/      | .__/ \__|\__, |
 |___/            |_|        |___/
```

A process manager that runs commands from a Procfile, each in its own pseudoterminal (PTY). Every process gets a real terminal, so you can attach to any running process and use interactive debuggers and REPLs directly, like `binding.irb`, `byebug`, `breakpoint`, etc.

Press `ctrl+]` to attach to a process and get a full interactive terminal session — type commands, inspect variables, step through code. Press `ctrl+]` again to detach and return to the combined output view.

## Usage

```bash
go-pty           # reads ./Procfile
go-pty -f FILE   # reads a specific Procfile
```

### Procfile format

```
web: bundle exec rails server -p 3000
worker: bundle exec sidekiq
css: tailwindcss --watch
```

### Keyboard shortcuts

| Mode | Key | Action |
|------|-----|--------|
| Normal | `ctrl+]` | Open process selection dialog |
| Normal | `ctrl+c` | Shut down all processes and exit |
| Dialog | `Up/Down` | Navigate process list |
| Dialog | `Enter` | Attach to selected process |
| Dialog | `Esc` | Cancel |
| Attached | `ctrl+]` | Detach and return to normal mode |

In **normal mode**, output from all processes is shown with colored prefixes. In **attached mode**, you get a raw terminal session with the selected process — full readline support, colors, and interactivity.

## Building

```bash
make build
make test
```

## License

MIT
