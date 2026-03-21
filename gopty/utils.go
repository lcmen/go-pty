package gopty

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	byteCtrlC       = 3  // ctrl+c
	byteCtrlR       = 18 // ctrl+r
	byteCtrlBracket = 29 // ctrl+]
	byteEsc         = '\x1b'
	byteEnter       = '\r'

	seqArrowUp   = "\x1b[A"
	seqArrowDown = "\x1b[B"
	seqEnter     = "\r"
	seqEsc       = "\x1b"
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

func FilterEntries(entries []Entry, names []string) ([]Entry, error) {
	if len(names) == 0 {
		return entries, nil
	}

	index := make(map[string]Entry, len(entries))
	for _, e := range entries {
		index[e.Name] = e
	}

	var result []Entry
	var unknown []string
	for _, name := range names {
		if e, ok := index[name]; ok {
			result = append(result, e)
		} else {
			unknown = append(unknown, name)
		}
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown service(s): %s", strings.Join(unknown, ", "))
	}

	return result, nil
}

func ParseEnvFile(path string) ([]Env, error) {
	var envs []Env
	err := readFile(path, "=", func(key, value string) {
		envs = append(envs, NewEnv(key, value))
	})
	if err != nil {
		return nil, err
	}
	return ExpandAll(envs), nil
}

func ParseProcfile(path string) ([]Entry, error) {
	var entries []Entry
	err := readFile(path, ":", func(key, value string) {
		entries = append(entries, Entry{
			Name:    strings.TrimSpace(key),
			Command: strings.TrimSpace(value),
		})
	})
	if err != nil {
		return nil, err
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

func readFile(path string, separator string, fn func(key, value string)) error {
	name := filepath.Base(path)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", name, err)
	}
	defer file.Close()

	var count int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, separator)
		if !ok {
			return fmt.Errorf("missing %s separator in: %q", separator, line)
		}
		fn(key, value)
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read %s: %w", name, err)
	}

	if count == 0 {
		return fmt.Errorf("no entry defined in %s", name)
	}

	return nil
}
