package gopty

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

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
	Color    string
	Entry
	ExitCode int
	cmd            *exec.Cmd
	done           chan struct{}
	master         *os.File
	mode     func() OutputMode
	prefixAll      string
	prefixAttached string
	reader         *bufio.Reader
}

func NewProcess(entry Entry, index int) *Process {
	color := colorPalette[index%len(colorPalette)]
	return &Process{
		Entry:          entry,
		Color:          color,
		done:           make(chan struct{}),
		prefixAll:      fmt.Sprintf("%s[%s]\033[0m", color, entry.Name),
		prefixAttached: fmt.Sprintf("%s[%s - attached]\033[0m", color, entry.Name),
	}
}

func (p *Process) Start() error {
	cmd := exec.Command("sh", "-c", p.Entry.Command)

	master, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start process %s: %w", p.Entry.Name, err)
	}

	p.cmd = cmd
	p.master = master
	p.reader = bufio.NewReader(master)

	return nil
}

func (p *Process) Monitor(w io.Writer) {
	p.read(w)

	exitCode, signal := p.exit()
	p.ExitCode = exitCode
	if signal != "" {
		fmt.Fprintf(w, "%s exited (signal %s)\r\n", p.prefix(), signal)
	} else {
		fmt.Fprintf(w, "%s exited (code %d)\r\n", p.prefix(), p.ExitCode)
	}
	close(p.done)
}

func (p *Process) Write(buf []byte) (int, error) {
	return p.master.Write(buf)
}

func (p *Process) Shutdown() {
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}

	// Send SIGINT for graceful shutdown
	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT)
}

func (p *Process) Kill(timeout time.Duration) {
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}

	// Wait for graceful exit, escalate to SIGKILL after timeout
	select {
	case <-p.done:
	case <-time.After(timeout):
		syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		<-p.done
	}
}

func (p *Process) exit() (int, string) {
	if err := p.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				return -1, ws.Signal().String()
			}
			return exitErr.ExitCode(), ""
		}
	}

	return 0, ""
}

func (p *Process) prefix() string {
	if p.mode() == OutputAttached {
		return p.prefixAttached
	}
	return p.prefixAll
}

func (p *Process) read(w io.Writer) {
	var err error
	var line []byte
	var n int

	buf := make([]byte, 4096)

	for {
		switch p.mode() {
		case OutputAll:
			// Read and write line by line
			line, err = p.reader.ReadBytes('\n')
			if len(line) > 0 {
				writeLine(w, p.prefix(), line)
			}

		case OutputAttached:
			// Read and write immediately to output
			n, err = p.reader.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
			}

		case OutputIgnored:
			// Read and discard to prevent child process from blocking
			n, err = p.reader.Read(buf)
		}

		if err != nil {
			break
		}
	}
}
