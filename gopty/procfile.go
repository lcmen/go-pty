package gopty

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Entry represents a single process definition from a Procfile
type Entry struct {
	Name    string
	Command string
}

// File represents a Procfile that has been read from disk
type File struct {
	lines []string
	path  string
}

// Open reads a Procfile from the given path and returns a File.
func Open(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Procfile: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read Procfile: %w", err)
	}

	return &File{
		lines: lines,
		path:  path,
	}, nil
}

// Parse parses the Procfile content into a slice of process entries.
// It expects lines in the format "name: command".
// Empty lines and lines starting with # are skipped.
func (f *File) Parse() ([]Entry, error) {
	var entries []Entry

	for lineNum, line := range f.lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip comment lines
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Split on first colon
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			return nil, fmt.Errorf("line %d: missing colon separator: %q", lineNum+1, line)
		}

		name := strings.TrimSpace(line[:colonIdx])
		command := strings.TrimSpace(line[colonIdx+1:])

		entries = append(entries, Entry{
			Name:    name,
			Command: command,
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no process definitions found in Procfile")
	}

	return entries, nil
}
