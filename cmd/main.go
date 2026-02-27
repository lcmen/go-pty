package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lcmen/go-pty/gopty"
)

var banner string = `
  __ _  ___        _ __ | |_ _   _
 / _` + "`" + ` |/ _ \ _____| '_ \| __| | | |
| (_| | (_) |_____| |_) | |_| |_| |
 \__, |\___/      | .__/ \__|\__, |
 |___/            |_|        |___/`

func main() {
	procfilePath := flag.String("f", "./Procfile", "path to Procfile")
	flag.Parse()

	m, err := initManager(*procfilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n\n", banner)
	fmt.Printf("Starting process(es) from %s:\n\n", *procfilePath)

	listenSig(func() {
		m.Shutdown()
	})

	m.Wait()
}

func initManager(path string) (*gopty.Manager, error) {
	pf, err := gopty.Open(path)
	if err != nil {
		return nil, err
	}

	entries, err := pf.Parse()
	if err != nil {
		return nil, err
	}

	m := gopty.NewManager(entries, os.Stdout)
	if err := m.StartAll(); err != nil {
		return nil, err
	}

	return m, nil
}

func listenSig(handler func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		handler()
	}()
}
