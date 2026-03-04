package gopty

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteLine(t *testing.T) {
	written := func(prefix, line string) string {
		var buf bytes.Buffer
		writeLine(&buf, prefix, []byte(line))
		return buf.String()
	}

	if diff := cmp.Diff("[web] hello\r\n", written("[web]", "hello\r\n")); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}

	if diff := cmp.Diff("[web] hello\r\n", written("[web]", "hello\n")); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}

	if diff := cmp.Diff("[web] hello\r\n", written("[web]", "hello\r")); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}

	if diff := cmp.Diff("[web] hello\r\n", written("[web]", "hello")); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}
}

func TestReadBytes(t *testing.T) {
	t.Run("reads up to n bytes", func(t *testing.T) {
		r := bytes.NewReader([]byte("hello"))
		buf, err := readBytes(r, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if diff := cmp.Diff("hel", string(buf)); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("reads fewer bytes if less available", func(t *testing.T) {
		r := bytes.NewReader([]byte("a"))
		buf, err := readBytes(r, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if diff := cmp.Diff("a", string(buf)); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})
}
