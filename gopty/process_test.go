package gopty

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewProcess(t *testing.T) {
	entry := Entry{Name: "web", Command: "bundle exec rails server"}
	p := NewProcess(entry, 0)

	if diff := cmp.Diff(entry, p.Entry); diff != "" {
		t.Errorf("Process.Entry mismatch (-expected +got):\n%s", diff)
	}

	if diff := cmp.Diff("\033[31m", p.Color); diff != "" {
		t.Errorf("Process.Color mismatch (-expected +got):\n%s", diff)
	}
}

func TestProcess_Read(t *testing.T) {
	stubProcess := func(entry Entry, mode OutputMode, input string, exitCodes ...int) *Process {
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

		p := NewProcess(entry, 0)
		p.cmd = cmd
		p.master = r
		p.outputMode = func() OutputMode { return mode }

		w.WriteString(input)
		w.Close()

		return p
	}

	t.Run("OutputAll prefixes each line", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAll, "line1\nline2\n")

		var buf bytes.Buffer
		p.Read(&buf)

		expected := "\033[31m[web]\033[0m line1\n\033[31m[web]\033[0m line2\n\033[31m[web]\033[0m exited (code 0)\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputAttached prints raw lines", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAttached, "line1\nline2\n")

		var buf bytes.Buffer
		p.Read(&buf)

		expected := "line1\nline2\n\033[31m[web]\033[0m exited (code 0)\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputIgnored drops output but prints exit", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputIgnored, "line1\nline2\n")

		var buf bytes.Buffer
		p.Read(&buf)

		expected := "\033[31m[web]\033[0m exited (code 0)\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("prints non-zero exit code", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAll, "hello\n", 1)

		var buf bytes.Buffer
		p.Read(&buf)

		expected := "\033[31m[web]\033[0m hello\n\033[31m[web]\033[0m exited (code 1)\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})
}
