package gopty

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestNewProcess(t *testing.T) {
	entry := Entry{Name: "web", Command: "bundle exec rails server"}
	p := NewProcess(entry, 0, io.Discard)

	if diff := cmp.Diff(entry, p.Entry); diff != "" {
		t.Errorf("Process.Entry mismatch (-expected +got):\n%s", diff)
	}

	if diff := cmp.Diff("\033[31m", p.Color); diff != "" {
		t.Errorf("Process.Color mismatch (-expected +got):\n%s", diff)
	}
}

func TestProcess_Monitor(t *testing.T) {
	stubProcess := func(entry Entry, mode OutputMode, input string, exitCodes ...int) (*Process, *bytes.Buffer) {
		r, w, _ := os.Pipe()

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

		var buf bytes.Buffer
		p := NewProcess(entry, 0, &buf)
		p.cmd = cmd
		p.pty = r
		p.reader = bufio.NewReader(r)
		p.mode = func() OutputMode { return mode }

		w.WriteString(input)
		w.Close()

		return p, &buf
	}

	t.Run("OutputAll prefixes each line", func(t *testing.T) {
		p, buf := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAll, "line1\nline2\n")
		p.Monitor()

		expected := "\033[31m[web]\033[0m line1\r\n\033[31m[web]\033[0m line2\r\n\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputAttached passes through raw bytes", func(t *testing.T) {
		p, buf := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAttached, "line1\nline2\n")
		p.Monitor()

		expected := "line1\nline2\n\033[31m[web - attached]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputIgnored drops output but prints exit", func(t *testing.T) {
		p, buf := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputIgnored, "line1\nline2\n")
		p.Monitor()

		expected := "\033[31m[web]\033[0m exited (code 0)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("prints non-zero exit code", func(t *testing.T) {
		p, buf := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAll, "hello\n", 1)
		p.Monitor()

		expected := "\033[31m[web]\033[0m hello\r\n\033[31m[web]\033[0m exited (code 1)\r\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})
}

func TestProcess_Shutdown(t *testing.T) {
	p := NewProcess(Entry{Name: "web", Command: "sleep 60"}, 0, io.Discard)
	p.mode = func() OutputMode { return OutputAll }

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	go p.Monitor()

	p.Shutdown()

	select {
	case <-p.done:
	case <-time.After(200 * time.Millisecond):
		t.Error("expected process to exit after SIGTERM")
	}
}

func TestProcess_Kill(t *testing.T) {
	// Use trap '' INT to ignore SIGINT
	p := NewProcess(Entry{Name: "web", Command: "trap '' INT; sleep 60"}, 0, io.Discard)
	p.mode = func() OutputMode { return OutputAll }

	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	go p.Monitor()

	// Send SIGINT directly (not via Shutdown)
	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT)

	p.Kill(200 * time.Millisecond)

	if p.ExitCode != -1 {
		t.Errorf("expected exit code -1 from SIGKILL, got %d", p.ExitCode)
	}
}

func TestProcess_Write(t *testing.T) {
	r, w, _ := os.Pipe()
	p := NewProcess(Entry{Name: "web", Command: "cmd"}, 0, io.Discard)
	p.pty = w

	p.Write([]byte("hello"))
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if diff := cmp.Diff("hello", buf.String()); diff != "" {
		t.Errorf("written data mismatch (-expected +got):\n%s", diff)
	}
}
