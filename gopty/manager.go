package gopty

import (
	"io"
	"sync"
	"sync/atomic"
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
