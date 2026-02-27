package gopty

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Entry struct {
	Name    string
	Command string
}

type File struct {
	lines []string
	path  string
}

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

func (f *File) Parse() ([]Entry, error) {
	var entries []Entry

	for _, line := range f.lines {
		if isComment(line) || isEmpty(line) {
			continue
		}

		name, command, err := extractNameAndCommand(line)
		if err != nil {
			return nil, err
		}

		entries = append(entries, Entry{
			Name:    name,
			Command: command,
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("No process defined in Procfile")
	}

	return entries, nil
}

func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

func isEmpty(line string) bool {
	return strings.TrimSpace(line) == ""
}

func extractNameAndCommand(line string) (string, string, error) {
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return "", "", fmt.Errorf("Missing colon separator in: %q", line)
	}

	name := strings.TrimSpace(line[:colonIdx])
	command := strings.TrimSpace(line[colonIdx+1:])

	return name, command, nil
}
