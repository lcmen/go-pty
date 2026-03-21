package gopty

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Process struct {
	Entry
	Color      string
	cmd        *exec.Cmd
	env        []Env
	mode       atomic.Value
	pty        *os.File
	ptyMu      sync.RWMutex
	terminated chan struct{}
}

func NewProcess(entry Entry, index int, env []Env) *Process {
	return &Process{
		Entry:      entry,
		Color:      ColorPalette[index%len(ColorPalette)],
		env:        env,
		terminated: make(chan struct{}),
	}
}

func (p *Process) Start() error {
	cmd := exec.Command("sh", "-c", p.Entry.Command)

	if p.env != nil {
		cmd.Env = os.Environ()
		for _, e := range p.env {
			cmd.Env = append(cmd.Env, e.Environ())
		}
	}

	master, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start process %s: %w", p.Entry.Name, err)
	}

	p.cmd = cmd
	p.pty = master

	return nil
}

func (p *Process) Stream(stdout io.Writer) error {
	defer p.close()

	prefix := fmt.Sprintf("%s[%s]\033[0m", p.Color, p.Name)
	p.read(stdout, prefix)

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
	unlock, err := p.lock(PtyWriteLock)
	if err != nil {
		return 0, err
	}
	defer unlock()
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
	unlock, err := p.lock(PtyReadLock)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return pty.GetsizeFull(p.pty)
}

func (p *Process) PtyResize(ws *pty.Winsize) error {
	unlock, err := p.lock(PtyWriteLock)
	if err != nil {
		return err
	}
	defer unlock()
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
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				return -1, ws.Signal().String()
			}
			return exitErr.ExitCode(), ""
		}
	}

	return 0, ""
}

func (p *Process) lock(mode ptyLockMode) (unlock func(), err error) {
	var u func()

	if mode == PtyReadLock {
		p.ptyMu.RLock()
		u = func() { p.ptyMu.RUnlock() }
	} else {
		p.ptyMu.Lock()
		u = func() { p.ptyMu.Unlock() }
	}

	if p.pty == nil {
		u()
		return nil, fmt.Errorf("pty not initialized")
	}
	return u, nil
}

func (p *Process) read(stdout io.Writer, prefix string) {
	unlock, err := p.lock(PtyReadLock)
	if err != nil {
		return
	}
	pty := p.pty
	unlock()

	reader := bufio.NewReader(pty)

	var line []byte
	var n int
	buf := make([]byte, 4096)

	for {
		mode, ok := p.mode.Load().(OutputMode)
		if !ok {
			mode = OutputAll
		}
		switch mode {
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
