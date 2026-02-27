package gopty

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

var colorPalette = []string{
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

type Process struct {
	Color string
	Entry
	cmd        *exec.Cmd
	master     *os.File
	outputMode func() OutputMode
}

func NewProcess(entry Entry, index int) *Process {
	return &Process{
		Entry: entry,
		Color: colorPalette[index%len(colorPalette)],
	}
}

func (p *Process) Start() error {
	cmd := exec.Command("sh", "-c", "exec "+p.Entry.Command)

	master, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start process %s: %w", p.Entry.Name, err)
	}

	p.cmd = cmd
	p.master = master

	return nil
}

func (p *Process) Read(w io.Writer) {
	scanner := bufio.NewScanner(p.master)
	for scanner.Scan() {
		line := scanner.Text()

		switch p.outputMode() {
		case OutputAll:
			fmt.Fprintf(w, "%s %s\n", p.prefix(), line)
		case OutputAttached:
			fmt.Fprintln(w, line)
		}
	}
}

func (p *Process) prefix() string {
	return fmt.Sprintf("%s[%s]\033[0m", p.Color, p.Entry.Name)
}
