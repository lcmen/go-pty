package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
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
	flag.Parse()

	fmt.Printf("%s\n\n", banner)
	fmt.Printf("Starting process(es) from %s:\n\n", *procfilePath)

	m, err := initManager(*procfilePath, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	restore := rawMode(os.Stdin)
	defer restore()

	c := gopty.NewController(m, os.Stdin, os.Stdout)
	listenResize(m.ResizeAll)
	listenTerm(c.Shutdown)

	go c.Run()
	m.Wait()
	c.Cleanup()
}

func initManager(path string, stdout io.Writer) (*gopty.Manager, error) {
	pf, err := gopty.Open(path)
	if err != nil {
		return nil, err
	}

	entries, err := pf.Parse()
	if err != nil {
		return nil, err
	}

	m := gopty.NewManager(entries, stdout)
	if err := m.StartAll(); err != nil {
		return nil, err
	}

	return m, nil
}

func listenResize(handler func(*pty.Winsize)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			ws, err := pty.GetsizeFull(os.Stdin)
			if err == nil {
				handler(ws)
			}
		}
	}()
}

func listenTerm(handler func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		handler()
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
