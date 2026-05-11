package gopty

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

type Command struct {
	Entry
	env []string
}

func NewCommand(entry Entry, env []string) *Command {
	return &Command{
		Entry: entry,
		env:   env,
	}
}

func (c *Command) Run(stdout io.Writer) error {
	cmd := exec.Command("sh", "-c", c.Command)
	cmd.Env = c.env

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("preflight %s failed: %w", c.Name, err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("preflight %s failed: %w", c.Name, err)
	}

	c.read(stdout, pipe)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("preflight %s failed: %w", c.Name, err)
	}

	return nil
}

func (c *Command) read(stdout io.Writer, output io.Reader) {
	reader := bufio.NewReader(output)
	prefix := fmt.Sprintf("[%s]", c.Name)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			fmt.Fprintf(stdout, "%s %s\r\n", prefix, bytes.TrimRight(line, "\r\n"))
		}
		if err != nil {
			break
		}
	}
}
