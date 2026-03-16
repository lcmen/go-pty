package gopty

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/creack/pty"
)

type Manager struct {
	processes   []*Process
	stdout      io.Writer
	terminating sync.Once
	wg          sync.WaitGroup
}

func NewManager(entries []Entry, stdout io.Writer) *Manager {
	m := &Manager{stdout: stdout}

	for i, entry := range entries {
		p := NewProcess(entry, i)
		p.mode.Store(OutputAll)
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
			if err := p.Stream(m.stdout); err != nil {
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

func (m *Manager) Attach(index int) (*Process, error) {
	if index < 0 || index >= len(m.processes) {
		return nil, fmt.Errorf("process index %d out of range [0, %d)", index, len(m.processes))
	}
	p := m.processes[index]
	m.updateAllModes(p)
	// Send enter to trigger prompt refresh so cursor lands at the correct position
	p.Write([]byte("\n"))
	return p, nil
}

func (m *Manager) Processes() []*Process {
	return m.processes
}

func (m *Manager) Detach() *Process {
	oldAttached := m.attached()
	m.updateAllModes(nil)
	return oldAttached
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
	p := m.attached()
	if p == nil {
		return 0, nil
	}
	return p.Write(buf)
}

func (m *Manager) attached() *Process {
	for _, p := range m.processes {
		mode, ok := p.mode.Load().(OutputMode)
		if ok && mode == OutputAttached {
			return p
		}
	}
	return nil
}

func (m *Manager) updateAllModes(attached *Process) {
	for _, p := range m.processes {
		switch {
		case attached == nil:
			p.mode.Store(OutputAll)
		case p == attached:
			p.mode.Store(OutputAttached)
		default:
			p.mode.Store(OutputIgnored)
		}
	}
}
