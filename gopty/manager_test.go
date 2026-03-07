package gopty

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Buffer that is goroutine safe for tests
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

func (sb *syncBuffer) ReadFrom(r io.Reader) (int64, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.ReadFrom(r)
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

	if diff := cmp.Diff(expected, m.processes, cmpopts.IgnoreUnexported(Process{})); diff != "" {
		t.Errorf("processes mismatch (-expected +got):\n%s", diff)
	}
}

func TestManager_Attach(t *testing.T) {
	t.Run("attaches process at valid index", func(t *testing.T) {
		m := NewManager([]Entry{
			{Name: "web", Command: "cmd1"},
		}, io.Discard)

		if err := m.Attach(0); err != nil {
			t.Errorf("Attach returned unexpected error: %v", err)
		}

		if m.Attached() != m.processes[0] {
			t.Error("attached should point to web process")
		}
	})

	t.Run("returns error for out-of-range index", func(t *testing.T) {
		m := NewManager([]Entry{
			{Name: "web", Command: "cmd1"},
		}, io.Discard)

		if err := m.Attach(5); err == nil {
			t.Error("Attach should return error for out-of-range index")
		}

		if err := m.Attach(-1); err == nil {
			t.Error("Attach should return error for negative index")
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

func TestManager_ResizeAll(t *testing.T) {
	var buf syncBuffer
	m := NewManager([]Entry{{Name: "web", Command: "sleep 60"}}, &buf)

	if err := m.StartAll(); err != nil {
		t.Fatalf("StartAll failed: %v", err)
	}

	ws := &pty.Winsize{Rows: 40, Cols: 100}
	m.ResizeAll(ws)

	got, err := m.processes[0].GetSize()
	if err != nil {
		t.Fatalf("GetsizeFull failed: %v", err)
	}
	m.Shutdown()
	if got.Rows != 40 || got.Cols != 100 {
		t.Errorf("expected 40x100, got %dx%d", got.Rows, got.Cols)
	}
}

func TestManager_Shutdown(t *testing.T) {
	t.Run("shuts down all processes", func(t *testing.T) {
		var buf syncBuffer
		m := NewManager([]Entry{
			{Name: "web", Command: "echo ready; trap 'exit 0' INT; sleep 60"},
			{Name: "worker", Command: "echo ready; trap 'exit 0' INT; sleep 60"},
		}, &buf)

		if err := m.StartAll(); err != nil {
			t.Fatalf("StartAll failed: %v", err)
		}

		waitFor(t, func() bool { return strings.Count(buf.String(), "ready") >= 2 })
		m.Shutdown()

		output := buf.String()
		if !strings.Contains(output, "[web]\033[0m exited") {
			t.Errorf("expected web exit message, got %q", output)
		}
		if !strings.Contains(output, "[worker]\033[0m exited") {
			t.Errorf("expected worker exit message, got %q", output)
		}
	})
}

func TestManager_StartAll(t *testing.T) {
	t.Run("monitors process output", func(t *testing.T) {
		var buf syncBuffer
		m := NewManager([]Entry{{Name: "web", Command: "echo hello"}}, &buf)

		if err := m.StartAll(); err != nil {
			t.Fatalf("StartAll failed: %v", err)
		}
		m.Wait()

		expected := "\033[31m[web]\033[0m hello\r\n\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("shuts down all processes when one crashes", func(t *testing.T) {
		var buf syncBuffer
		m := NewManager([]Entry{
			{Name: "web", Command: "echo ready; sleep 60"},
			{Name: "worker", Command: "echo ready; exit 1"},
		}, &buf)

		if err := m.StartAll(); err != nil {
			t.Fatalf("StartAll failed: %v", err)
		}
		m.Wait()

		output := buf.String()
		if !strings.Contains(output, "[worker]\033[0m exited (code 1)") {
			t.Errorf("expected worker crash message, got %q", output)
		}
		if !strings.Contains(output, "[web]\033[0m exited (signal interrupt)") {
			t.Errorf("expected web exit message, got %q", output)
		}
	})

	t.Run("does not shut down when process exits cleanly", func(t *testing.T) {
		var buf syncBuffer
		m := NewManager([]Entry{
			{Name: "web", Command: "echo ready; sleep 60"},
			{Name: "worker", Command: "echo ready; exit 0"},
		}, &buf)

		if err := m.StartAll(); err != nil {
			t.Fatalf("StartAll failed: %v", err)
		}

		waitFor(t, func() bool { return strings.Contains(buf.String(), "exited (code 0)") })

		if strings.Contains(buf.String(), "[web]\033[0m exited") {
			t.Error("web should still be running after worker's clean exit")
		}

		m.Shutdown()
	})
}

func TestManager_WriteToAttached(t *testing.T) {
	t.Run("forwards bytes to attached process", func(t *testing.T) {
		r, w, _ := os.Pipe()
		m := NewManager([]Entry{{Name: "web", Command: "cmd"}}, io.Discard)
		m.processes[0].pty = w
		m.Attach(0)

		m.WriteToAttached([]byte("hello"))
		w.Close()

		var buf syncBuffer
		buf.ReadFrom(r)
		// Expect '\n' from Attach(), then 'hello'
		if buf.String() != "\nhello" {
			t.Errorf("expected %q, got %q", "\nhello", buf.String())
		}
	})

	t.Run("no-op when no process is attached", func(t *testing.T) {
		m := NewManager([]Entry{{Name: "web", Command: "cmd"}}, io.Discard)

		n, err := m.WriteToAttached([]byte("hello"))

		if n != 0 || err != nil {
			t.Errorf("expected (0, nil), got (%d, %v)", n, err)
		}
	})
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 2s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
