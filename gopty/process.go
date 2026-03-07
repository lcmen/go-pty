package gopty

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
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
	Color string
	Entry
	cmd    *exec.Cmd
	done   chan struct{}
	mode   func() OutputMode
	prefix string
	pty    *os.File
	ptyMu  sync.RWMutex
	reader *bufio.Reader
	stdout io.Writer
}

func NewProcess(entry Entry, index int) *Process {
	color := colorPalette[index%len(colorPalette)]
	return &Process{
		Entry:  entry,
		Color:  color,
		done:   make(chan struct{}),
		prefix: fmt.Sprintf("%s[%s]\033[0m", color, entry.Name),
	}
}

func (p *Process) Start() error {
	cmd := exec.Command("sh", "-c", p.Entry.Command)

	master, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start process %s: %w", p.Entry.Name, err)
	}

	p.cmd = cmd
	p.pty = master
	p.reader = bufio.NewReader(master)

	return nil
}

func (p *Process) Monitor() error {
	defer p.close()
	p.read()

	exitCode, signal := p.exit()

	if signal != "" {
		fmt.Fprintf(p.stdout, "%s exited (signal %s)\r\n", p.prefix, signal)
	} else {
		fmt.Fprintf(p.stdout, "%s exited (code %d)\r\n", p.prefix, exitCode)
	}

	// Notify that process exited and we don't need to send SIGKILL
	close(p.done)

	if exitCode != 0 {
		return fmt.Errorf("process %s exited with code %d", p.Name, exitCode)
	}

	return nil
}

func (p *Process) Write(buf []byte) (int, error) {
	p.ptyMu.RLock()
	defer p.ptyMu.RUnlock()
	if p.pty == nil {
		return 0, fmt.Errorf("pty not initialized")
	}
	return p.pty.Write(buf)
}

func (p *Process) Shutdown(timeout time.Duration) {
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}

	// Send SIGINT for graceful shutdown
	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT)

	// Wait for graceful exit, escalate to SIGKILL after timeout
	select {
	case <-p.done:
	case <-time.After(timeout):
		syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		<-p.done
	}
}

func (p *Process) PtySize() (*pty.Winsize, error) {
	p.ptyMu.RLock()
	defer p.ptyMu.RUnlock()
	if p.pty == nil {
		return nil, fmt.Errorf("pty not initialized")
	}
	return pty.GetsizeFull(p.pty)
}

func (p *Process) PtyResize(ws *pty.Winsize) error {
	p.ptyMu.Lock()
	defer p.ptyMu.Unlock()
	if p.pty == nil {
		return fmt.Errorf("pty not initialized")
	}
	return pty.Setsize(p.pty, ws)
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

func (p *Process) read() {
	var err error
	var line []byte
	var n int

	buf := make([]byte, 4096)

	for {
		switch p.mode() {
		case OutputAll:
			// Read and write line by line
			line, err = p.reader.ReadBytes('\n')
			if len(line) > 0 && err == nil {
				fmt.Fprintf(p.stdout, "%s %s\r\n", p.prefix, bytes.TrimRight(line, "\r\n"))
			}

		case OutputAttached:
			// Read and write immediately to output
			n, err = p.reader.Read(buf)
			if n > 0 && err == nil {
				p.stdout.Write(buf[:n])
			}

		case OutputIgnored:
			// Read and discard to prevent child process from blocking
			_, err = p.reader.Read(buf)
		}

		if err != nil {
			break
		}
	}
}

func (p *Process) close() error {
	p.ptyMu.Lock()
	defer p.ptyMu.Unlock()
	if p.pty == nil {
		return nil
	}
	err := p.pty.Close()
	p.pty = nil
	return err
}
