# go-pty

A terminal multiplexer that runs multiple commands concurrently, each in its own pseudoterminal. Think [foreman](https://github.com/ddollar/foreman) or [overmind](https://github.com/DarthSim/overmind), but simpler — prefixed log output by default, with the ability to step into any process for full interactive terminal access.

This makes it possible to use interactive debuggers (`binding.pry`, `byebug`, `pdb`, `debugger`) inside a Procfile-managed dev environment.

## Usage

```bash
go-pty              # reads ./Procfile
go-pty Procfile.dev # reads a specific Procfile
```

### Procfile format

Standard Heroku/foreman format:

```
# Comments start with #
web: bundle exec rails server -p 3000
worker: bundle exec sidekiq
css: tailwindcss --watch
```

### Modes

**All Output Mode** (default) — prefixed, color-coded output from all processes:

```
[web]    Starting server on port 3000
[worker] Sidekiq 7.0.0 starting...
[css]    Rebuilding...
```

Commands:
- `!1` or `!web` + Enter — step into a process (by number or name)
- `ctrl+c` — kill all processes and exit

**Stepped In Mode** — full interactive terminal access to one process:

- All keystrokes forwarded to the attached process
- Other processes' output is buffered (up to 1MB each)
- `ctrl+]` — detach and return to all output mode

## Building

```bash
go build -o go-pty ./cmd/go-pty
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [github.com/creack/pty](https://github.com/creack/pty) | Spawn processes in a pseudoterminal |
| [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) | Raw mode, terminal state save/restore |

## License

MIT
