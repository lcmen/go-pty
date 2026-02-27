package gopty

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var cmpOpt = cmpopts.IgnoreUnexported(Process{})

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func TestNewManager(t *testing.T) {
	m := NewManager([]Entry{
		{Name: "web", Command: "echo hello"},
		{Name: "worker", Command: "echo world"},
	}, io.Discard)

	expected := []*Process{
		{Entry: Entry{Name: "web", Command: "echo hello"}, Color: "\033[31m"},
		{Entry: Entry{Name: "worker", Command: "echo world"}, Color: "\033[32m"},
	}

	if diff := cmp.Diff(expected, m.processes, cmpOpt); diff != "" {
		t.Errorf("processes mismatch (-expected +got):\n%s", diff)
	}
}

func TestManager_Attach(t *testing.T) {
	t.Run("returns true for existing process", func(t *testing.T) {
		m := NewManager([]Entry{
			{Name: "web", Command: "cmd1"},
		}, io.Discard)

		if diff := cmp.Diff(true, m.Attach("web")); diff != "" {
			t.Errorf("Attach result mismatch (-expected +got):\n%s", diff)
		}

		if m.Attached() != m.processes[0] {
			t.Error("attached should point to web process")
		}
	})

	t.Run("returns false for nonexistent process", func(t *testing.T) {
		m := NewManager([]Entry{
			{Name: "web", Command: "cmd1"},
		}, io.Discard)

		if diff := cmp.Diff(false, m.Attach("nonexistent")); diff != "" {
			t.Errorf("Attach result mismatch (-expected +got):\n%s", diff)
		}

		if m.Attached() != nil {
			t.Error("Attach should attach any process")
		}
	})
}

func TestManager_Detach(t *testing.T) {
	m := NewManager([]Entry{
		{Name: "web", Command: "cmd1"},
	}, io.Discard)

	m.attached.Store(m.processes[0])
	m.Detach()

	if m.Attached() != nil {
		t.Error("attached should be nil after Detach")
	}
}

func TestManager_StartAll(t *testing.T) {
	var buf bytes.Buffer
	m := NewManager([]Entry{{Name: "web", Command: "echo hello"}}, &buf)

	if err := m.StartAll(); err != nil {
		t.Fatalf("StartAll failed: %v", err)
	}
	m.Wait()

	expected := "\033[31m[web]\033[0m hello\n\033[31m[web]\033[0m exited (code 0)\n"
	if diff := cmp.Diff(expected, buf.String()); diff != "" {
		t.Errorf("output mismatch (-expected +got):\n%s", diff)
	}
}

func TestManager_Shutdown(t *testing.T) {
	var buf syncBuffer
	m := NewManager([]Entry{
		{Name: "web", Command: "sleep 60"},
		{Name: "worker", Command: "sleep 60"},
	}, &buf)

	if err := m.StartAll(); err != nil {
		t.Fatalf("StartAll failed: %v", err)
	}

	m.Shutdown()

	output := buf.String()
	if !strings.Contains(output, "[web]\033[0m exited (code") {
		t.Errorf("expected web exit message, got %q", output)
	}
	if !strings.Contains(output, "[worker]\033[0m exited (code") {
		t.Errorf("expected worker exit message, got %q", output)
	}
}
