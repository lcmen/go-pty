package gopty

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	byteCtrlC       = 3  // ctrl+c
	byteCtrlBracket = 29 // ctrl+]
	byteEsc         = '\x1b'
	byteEnter       = '\r'

	seqArrowUp   = "\x1b[A"
	seqArrowDown = "\x1b[B"
)

var ColorPalette = []string{
	"\033[31m", // red
	"\033[32m", // green
	"\033[33m", // yellow
	"\033[34m", // blue
	"\033[35m", // magenta
	"\033[36m", // cyan
	"\033[91m", // bright red
	"\033[92m", // bright green
	"\033[93m", // bright yellow
	"\033[94m", // bright blue
	"\033[95m", // bright magenta
	"\033[96m", // bright cyan
}

type OutputMode int

const (
	OutputAll OutputMode = iota
	OutputAttached
	OutputIgnored
)

type ptyLockMode int

const (
	PtyReadLock ptyLockMode = iota
	PtyWriteLock
)

type Entry struct {
	Name    string
	Command string
}

func ParseProcfile(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Procfile: %w", err)
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		name, command, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("missing colon separator in: %q", line)
		}

		entries = append(entries, Entry{
			Name:    strings.TrimSpace(name),
			Command: strings.TrimSpace(command),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read Procfile: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no process defined in Procfile")
	}

	return entries, nil
}

func readByte(r io.Reader) (byte, error) {
	var buf [1]byte
	_, err := r.Read(buf[:])
	return buf[0], err
}

func readBytes(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	read, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:read], nil
}
