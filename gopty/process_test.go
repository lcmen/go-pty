package gopty

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestNewProcess(t *testing.T) {
	p := NewProcess(Entry{Name: "web", Command: "bundle exec rails server"}, 0)

	if diff := cmp.Diff("web", p.Name); diff != "" {
		t.Errorf("Process.Name mismatch (-expected +got):\n%s", diff)
	}

	if diff := cmp.Diff("\033[31m", p.Color); diff != "" {
		t.Errorf("Process.Color mismatch (-expected +got):\n%s", diff)
	}
}

func TestProcess_Start(t *testing.T) {
	p := NewProcess(Entry{Name: "web", Command: "echo hello"}, 0)

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if p.cmd == nil || p.cmd.Process == nil {
		t.Fatal("expected cmd.Process to be set after Start")
	}

	if p.pty == nil {
		t.Fatal("expected pty to be set after Start")
	}
}

func TestProcess_Stream(t *testing.T) {
	t.Run("OutputAll prefixes each line", func(t *testing.T) {
		var buf bytes.Buffer
		p, w := stubProcess(t, "line1\nline2\n", 0)
		w.Close()

		p.Stream(&buf)

		expected := "\033[31m[web]\033[0m line1\r\n\033[31m[web]\033[0m line2\r\n\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputAttached passes through raw bytes", func(t *testing.T) {
		var buf bytes.Buffer
		p, w := stubProcess(t, "line1\nline2\n", 0)
		p.mode.Store(OutputAttached)
		w.Close()

		p.Stream(&buf)

		expected := "line1\nline2\n\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputIgnored drops output but prints exit", func(t *testing.T) {
		var buf bytes.Buffer
		p, w := stubProcess(t, "line1\nline2\n", 0)
		p.mode.Store(OutputIgnored)
		w.Close()

		p.Stream(&buf)

		expected := "\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("prints non-zero exit code", func(t *testing.T) {
		var buf bytes.Buffer
		p, w := stubProcess(t, "hello\n", 1)
		w.Close()

		p.Stream(&buf)

		expected := "\033[31m[web]\033[0m hello\r\n\033[31m[web]\033[0m exited (code 1)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})
}

func TestProcess_Shutdown(t *testing.T) {
	p := stubProcessWithCommand(t, "sleep 60")

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	go p.Stream(io.Discard)

	p.Shutdown(200 * time.Millisecond)

	select {
	case <-p.terminated:
	case <-time.After(300 * time.Millisecond):
		t.Error("expected process to exit after shutdown")
	}
}

func TestProcess_Write(t *testing.T) {
	r, w, _ := os.Pipe()
	p := NewProcess(Entry{Name: "web", Command: "cmd"}, 0)
	p.pty = w

	p.Write([]byte("hello"))
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if diff := cmp.Diff("hello", buf.String()); diff != "" {
		t.Errorf("written data mismatch (-expected +got):\n%s", diff)
	}
}

// stubProcess creates a test process with mocked PTY for Stream tests.
// Returns the process and the write end of the PTY pipe.
func stubProcess(t *testing.T, input string, exitCode int) (*Process, *os.File) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	p := stubProcessWithCommand(t, "cmd")
	p.pty = r

	// Create command that exits with specified code
	if exitCode == 0 {
		p.cmd = exec.Command("true")
	} else {
		p.cmd = exec.Command("sh", "-c", fmt.Sprintf("exit %d", exitCode))
	}
	p.cmd.Start()

	return p, w
}

// stubProcessWithCommand creates a test process with a real command for lifecycle tests.
// Use this for tests that need actual process execution (e.g., Shutdown tests).
func stubProcessWithCommand(t *testing.T, command string) *Process {
	t.Helper()

	p := NewProcess(Entry{Name: "web", Command: command}, 0)
	p.mode.Store(OutputAll)
	return p
}
