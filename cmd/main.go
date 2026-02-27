package main

import (
	"fmt"
	"os"

	"github.com/lcmen/go-pty/gopty"
)

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

	// Print entries (sanity check for Phase 1)
	fmt.Println(`
  __ _  ___        _ __ | |_ _   _
 / _` + "`" + ` |/ _ \ _____| '_ \| __| | | |
| (_| | (_) |_____| |_) | |_| |_| |
 \__, |\___/      | .__/ \__|\__, |
 |___/            |_|        |___/`)
	fmt.Printf("\nStarting %d process(es) from %s:\n\n", len(entries), procfilePath)
	for _, entry := range entries {
		fmt.Printf("[%s] %s\n", entry.Name, entry.Command)
	}
}
