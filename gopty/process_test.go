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
		p := stubProcess(t, "line1\nline2\n")
		p.Stream(&buf, func() OutputMode { return OutputAll })

		expected := "\033[31m[web]\033[0m line1\r\n\033[31m[web]\033[0m line2\r\n\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputAttached passes through raw bytes", func(t *testing.T) {
		var buf bytes.Buffer
		p := stubProcess(t, "line1\nline2\n")
		p.Stream(&buf, func() OutputMode { return OutputAttached })

		expected := "line1\nline2\n\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputIgnored drops output but prints exit", func(t *testing.T) {
		var buf bytes.Buffer
		p := stubProcess(t, "line1\nline2\n")
		p.Stream(&buf, func() OutputMode { return OutputIgnored })

		expected := "\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("prints non-zero exit code", func(t *testing.T) {
		var buf bytes.Buffer
		p := stubProcess(t, "hello\n", 1)
		p.Stream(&buf, func() OutputMode { return OutputAll })

		expected := "\033[31m[web]\033[0m hello\r\n\033[31m[web]\033[0m exited (code 1)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})
}

func TestProcess_Shutdown(t *testing.T) {
	p := NewProcess(Entry{Name: "web", Command: "sleep 60"}, 0)

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	go p.Stream(io.Discard, func() OutputMode { return OutputAll })

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

func stubProcess(t *testing.T, input string, exitCodes ...int) *Process {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}

	exitCode := 0
	if len(exitCodes) > 0 {
		exitCode = exitCodes[0]
	}

	var cmd *exec.Cmd
	if exitCode == 0 {
		cmd = exec.Command("true")
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("exit %d", exitCode))
	}
	cmd.Start()

	p := NewProcess(Entry{Name: "web", Command: "cmd"}, 0)
	p.cmd = cmd
	p.pty = r

	w.WriteString(input)
	w.Close()

	return p
}
