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
	cmd        *exec.Cmd
	terminated chan struct{}
	pty        *os.File
	ptyMu      sync.RWMutex
}

func NewProcess(entry Entry, index int) *Process {
	return &Process{
		Entry:      entry,
		Color:      colorPalette[index%len(colorPalette)],
		terminated: make(chan struct{}),
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

	return nil
}

func (p *Process) Monitor(stdout io.Writer, mode func() OutputMode) error {
	defer p.close()

	prefix := fmt.Sprintf("%s[%s]\033[0m", p.Color, p.Name)
	p.read(stdout, mode, prefix)

	// Notify that process exited and we don't need to send SIGKILL
	close(p.terminated)

	code, signal := p.exitStatus()

	if signal != "" {
		fmt.Fprintf(stdout, "%s exited (signal %s)\r\n", prefix, signal)
	} else {
		fmt.Fprintf(stdout, "%s exited (code %d)\r\n", prefix, code)
	}

	if code != 0 {
		return fmt.Errorf("process %s exited with code %d", p.Name, code)
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
	case <-p.terminated:
	case <-time.After(timeout):
		syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		<-p.terminated
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

func (p *Process) exitStatus() (int, string) {
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

func (p *Process) read(stdout io.Writer, mode func() OutputMode, prefix string) {
	reader := bufio.NewReader(p.pty)

	var err error
	var line []byte
	var n int

	buf := make([]byte, 4096)

	for {
		switch mode() {
		case OutputAll:
			// Read and write line by line
			line, err = reader.ReadBytes('\n')
			if len(line) > 0 && err == nil {
				fmt.Fprintf(stdout, "%s %s\r\n", prefix, bytes.TrimRight(line, "\r\n"))
			}

		case OutputAttached:
			// Read and write immediately to output
			n, err = reader.Read(buf)
			if n > 0 && err == nil {
				stdout.Write(buf[:n])
			}

		case OutputIgnored:
			// Read and discard to prevent child process from blocking
			_, err = reader.Read(buf)
		}

		if err != nil {
			break
		}
	}
}
