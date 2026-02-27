package gopty

import (
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/creack/pty"
)

type OutputMode int

const (
	OutputAll OutputMode = iota
	OutputAttached
	OutputIgnored
)

type Manager struct {
	attached  atomic.Pointer[Process]
	out       io.Writer
	processes []*Process
	wg        sync.WaitGroup
}

func NewManager(entries []Entry, out io.Writer) *Manager {
	m := &Manager{out: out}

	for i, entry := range entries {
		p := NewProcess(entry, i)
		p.outputMode = m.outputMode(p)
		m.processes = append(m.processes, p)
	}

	return m
}

func (m *Manager) StartAll() error {
	for _, p := range m.processes {
		if err := p.Start(); err != nil {
			return err
		}

		m.wg.Go(func() {
			p.Read(m.out)
		})
	}

	// Initial size
	if f, ok := m.out.(*os.File); ok {
		if ws, err := pty.GetsizeFull(f); err == nil {
			m.ResizeAll(ws)
		}
	}

	return nil
}

func (m *Manager) Attached() *Process {
	return m.attached.Load()
}

func (m *Manager) Attach(name string) bool {
	for _, p := range m.processes {
		if p.Entry.Name == name {
			m.attached.Store(p)
			return true
		}
	}
	return false
}

func (m *Manager) Detach() {
	m.attached.Store(nil)
}

func (m *Manager) Shutdown() {
	for _, p := range m.processes {
		if p.cmd != nil && p.cmd.Process != nil {
			p.cmd.Process.Signal(syscall.SIGTERM)
		}
	}
	m.wg.Wait()
}

func (m *Manager) ResizeAll(ws *pty.Winsize) {
	for _, p := range m.processes {
		if p.master != nil {
			pty.Setsize(p.master, ws)
		}
	}
}

func (m *Manager) Wait() {
	m.wg.Wait()
}

func (m *Manager) outputMode(p *Process) func() OutputMode {
	return func() OutputMode {
		attached := m.Attached()
		if attached == nil {
			return OutputAll
		}
		if attached == p {
			return OutputAttached
		}
		return OutputIgnored
	}
}
