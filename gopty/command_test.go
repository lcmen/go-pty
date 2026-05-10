package gopty

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommand_Run(t *testing.T) {
	t.Run("runs command and writes output", func(t *testing.T) {
		var buf bytes.Buffer
		c := NewCommand(Entry{Name: "_", Command: "echo preflight-ok"}, nil)

		if err := c.Run(&buf); err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if !strings.Contains(buf.String(), "preflight-ok") {
			t.Fatalf("expected output to contain preflight message, got %q", buf.String())
		}
	})

	t.Run("normalizes line endings", func(t *testing.T) {
		var buf bytes.Buffer
		c := NewCommand(Entry{Name: "_", Command: "printf 'one\\ntwo\\n'"}, nil)

		if err := c.Run(&buf); err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if got, expected := buf.String(), "[_] one\r\n[_] two\r\n"; got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("does not double existing CRLF line endings", func(t *testing.T) {
		var buf bytes.Buffer
		c := NewCommand(Entry{Name: "_", Command: "printf 'one\\r\\ntwo\\r\\n'"}, nil)

		if err := c.Run(&buf); err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if got, expected := buf.String(), "[_] one\r\n[_] two\r\n"; got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("prints partial final line", func(t *testing.T) {
		var buf bytes.Buffer
		c := NewCommand(Entry{Name: "_", Command: "printf 'one'"}, nil)

		if err := c.Run(&buf); err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if got, expected := buf.String(), "[_] one\r\n"; got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("returns exit code error", func(t *testing.T) {
		c := NewCommand(Entry{Name: "_", Command: "exit 7"}, nil)

		err := c.Run(&bytes.Buffer{})
		if err == nil {
			t.Fatal("expected Run to fail")
		}
		if !strings.Contains(err.Error(), "preflight _ failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("passes environment variables", func(t *testing.T) {
		var buf bytes.Buffer
		c := NewCommand(
			Entry{Name: "_env", Command: "echo $FOO"},
			[]Env{{Key: "FOO", Value: "bar"}},
		)

		if err := c.Run(&buf); err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if !strings.Contains(buf.String(), "bar") {
			t.Fatalf("expected output to contain env var value, got %q", buf.String())
		}
	})
}
