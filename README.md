# go-pty

A terminal multiplexer that runs multiple commands concurrently, each in its own pseudoterminal. Think [foreman](https://github.com/ddollar/foreman) or [overmind](https://github.com/DarthSim/overmind), but simpler — prefixed log output by default, with the ability to attach to any process for full interactive terminal access.

This makes it possible to use interactive debuggers (`binding.pry`, `byebug`, `pdb`, `debugger`) inside a Procfile-managed dev environment.

## Usage

```bash
go-pty           # reads ./Procfile
go-pty -f FILE   # reads a specific Procfile
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

**Normal** (default) — prefixed, color-coded output from all processes:

```
[web]    Starting server on port 3000
[worker] Sidekiq 7.0.0 starting...
[css]    Rebuilding...
```

- `ctrl+]` — open process selection dialog
- `ctrl+c` — shut down all processes and exit

**Dialog** — pick a process to attach to:

```
Select a process (↑/↓ navigate, Enter select, Esc cancel):

  > 1. web
    2. worker
    3. css
```

- Arrow keys — navigate the list
- `Enter` — attach to the highlighted process
- `Esc` — cancel, return to normal mode

**Attached** — output from the attached process, prefixed with `[name - attached]`:

```
[web - attached] Started GET "/" for 127.0.0.1
[web - attached] Processing by HomeController#index as HTML
```

- `ctrl+]` — detach and return to normal mode

## Building

```bash
make build
```

## Development

```bash
make test    # Run tests
make lint    # Run go vet and go fmt
make clean   # Remove built binary
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [github.com/creack/pty](https://github.com/creack/pty) | Spawn processes in pseudoterminals |
| [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) | Raw mode, terminal state save/restore |
| [github.com/google/go-cmp](https://github.com/google/go-cmp) | Test assertions (dev only) |

## License

MIT
