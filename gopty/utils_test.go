package gopty

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
