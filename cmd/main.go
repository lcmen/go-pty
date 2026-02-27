package main

import (
	"fmt"
	"os"

	"github.com/lcmen/go-pty/gopty"
)

var banner string = `
  __ _  ___        _ __ | |_ _   _
 / _` + "`" + ` |/ _ \ _____| '_ \| __| | | |
| (_| | (_) |_____| |_) | |_| |_| |
 \__, |\___/      | .__/ \__|\__, |
 |___/            |_|        |___/`

func main() {
	var procfilePath string

	if len(os.Args) > 1 {
		procfilePath = os.Args[1]
	} else {
		procfilePath = "./Procfile"
	}

	pf, err := gopty.Open(procfilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	entries, err := pf.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n\n", banner)
	fmt.Printf("Starting %d process(es) from %s:\n\n", len(entries), procfilePath)

	m := gopty.NewManager(entries, os.Stdout)
	if err := m.StartAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	m.Wait()
}
