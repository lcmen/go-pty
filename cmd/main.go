package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/lcmen/go-pty/gopty"
	"golang.org/x/term"
)

var banner string = `
  __ _  ___        _ __ | |_ _   _
 / _` + "`" + ` |/ _ \ _____| '_ \| __| | | |
| (_| | (_) |_____| |_) | |_| |_| |
 \__, |\___/      | .__/ \__|\__, |
 |___/            |_|        |___/`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go-pty [options]\n\nProcess manager that runs commands from a Procfile with PTY support.\n\nOptions:\n")
		flag.PrintDefaults()
	}

	procfilePath := flag.String("f", "./Procfile", "path to Procfile")
	serviceFilter := flag.String("s", "", "comma-separated list of services to run (e.g. web,worker)")
	flag.Parse()

	fmt.Printf("%s\n\n", banner)
	if *serviceFilter != "" {
		fmt.Printf("Starting %s process(es):\n\n", *serviceFilter)
	} else {
		fmt.Printf("Starting all process(es):\n\n")
	}

	entries, err := parseEntries(*procfilePath, *serviceFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	m, err := initManager(entries, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	restore := rawMode(os.Stdin)

	defer cancel()
	defer restore()

	c := gopty.NewController(m, os.Stdin, os.Stdout)
	listenResize(ctx, m.ResizeAll)
	listenTerm(ctx, c.Shutdown)

	go c.Run()
	m.WaitAll()
	c.Cleanup()
}

func parseEntries(path, filter string) ([]gopty.Entry, error) {
	entries, err := gopty.ParseProcfile(path)
	if err != nil {
		return nil, err
	}

	if filter == "" {
		return entries, nil
	}

	names := strings.Split(filter, ",")
	for i, n := range names {
		names[i] = strings.TrimSpace(n)
	}
	return gopty.FilterEntries(entries, names)
}

func initManager(entries []gopty.Entry, stdout io.Writer) (*gopty.Manager, error) {
	m := gopty.NewManager(entries, stdout)
	if err := m.StartAll(); err != nil {
		return nil, err
	}

	return m, nil
}

func listenResize(ctx context.Context, handler func(*pty.Winsize)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigCh:
				ws, err := pty.GetsizeFull(os.Stdin)
				if err == nil {
					handler(ws)
				}
			}
		}
	}()
}

func listenTerm(ctx context.Context, handler func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(sigCh)
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			handler()
		}
	}()
}

func rawMode(f *os.File) func() {
	oldState, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error entering raw mode: %v\n", err)
		os.Exit(1)
	}
	return func() {
		term.Restore(int(f.Fd()), oldState)
	}
}
