package gopty

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
)

type OutputMode int

const (
	OutputAll OutputMode = iota
	OutputAttached
	OutputIgnored
)

type Manager struct {
	attached    atomic.Pointer[Process]
	processes   []*Process
	stdout      io.Writer
	terminating sync.Once
	wg          sync.WaitGroup
}

func NewManager(entries []Entry, stdout io.Writer) *Manager {
	m := &Manager{stdout: stdout}

	for i, entry := range entries {
		p := NewProcess(entry, i)
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
			// If one of the process crashed, shutdown the whole manager
			if err := p.Monitor(m.stdout, m.mode(p)); err != nil {
				m.Shutdown()
			}
		})
	}

	// Set the initial size for all PTYs
	if f, ok := m.stdout.(*os.File); ok {
		if ws, err := pty.GetsizeFull(f); err == nil {
			m.ResizeAll(ws)
		}
	}

	return nil
}

func (m *Manager) Attached() *Process {
	return m.attached.Load()
}

func (m *Manager) Attach(index int) error {
	if index < 0 || index >= len(m.processes) {
		return fmt.Errorf("process index %d out of range [0, %d)", index, len(m.processes))
	}
	m.attached.Store(m.processes[index])
	// Send enter to trigger prompt refresh so cursor lands at the correct position
	m.WriteToAttached([]byte("\n"))
	return nil
}

func (m *Manager) Processes() []*Process {
	return m.processes
}

func (m *Manager) Detach() {
	m.attached.Store(nil)
}

func (m *Manager) Shutdown() {
	m.terminating.Do(func() {
		timeout := 5 * time.Second
		var wg sync.WaitGroup
		for _, p := range m.processes {
			wg.Go(func() {
				p.Shutdown(timeout)
			})
		}
		wg.Wait()
	})
}

func (m *Manager) ResizeAll(ws *pty.Winsize) {
	for _, p := range m.processes {
		p.PtyResize(ws)
	}
}

func (m *Manager) WaitAll() {
	m.wg.Wait()
}

func (m *Manager) WriteToAttached(buf []byte) (int, error) {
	p := m.Attached()
	if p == nil {
		return 0, nil
	}
	return p.Write(buf)
}

func (m *Manager) mode(p *Process) func() OutputMode {
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
