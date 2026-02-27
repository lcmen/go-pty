package gopty

import (
	"bytes"
	"os"
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
	stubProcess := func(entry Entry, mode OutputMode, input string) *Process {
		r, w, _ := os.Pipe()

		p := NewProcess(entry, 0)
		p.outputMode = func() OutputMode { return mode }
		p.master = r

		w.WriteString(input)
		w.Close()

		return p
	}

	t.Run("OutputAll prefixes each line", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAll, "line1\nline2\n")

		var buf bytes.Buffer
		p.Read(&buf)

		expected := "\033[31m[web]\033[0m line1\n\033[31m[web]\033[0m line2\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputAttached prints raw lines", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputAttached, "line1\nline2\n")

		var buf bytes.Buffer
		p.Read(&buf)

		expected := "line1\nline2\n"
		if diff := cmp.Diff(expected, buf.String()); diff != "" {
			t.Errorf("output mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("OutputIgnored drops all output", func(t *testing.T) {
		p := stubProcess(Entry{Name: "web", Command: "cmd"}, OutputIgnored, "line1\nline2\n")

		var buf bytes.Buffer
		p.Read(&buf)

		if buf.String() != "" {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})
}
