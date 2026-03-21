package gopty

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilterEntries(t *testing.T) {
	entries := []Entry{
		{Name: "web", Command: "rails server"},
		{Name: "worker", Command: "sidekiq"},
		{Name: "clock", Command: "clockwork"},
	}

	t.Run("filters entries to matching names", func(t *testing.T) {
		got, err := FilterEntries(entries, []string{"web", "worker"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []Entry{
			{Name: "web", Command: "rails server"},
			{Name: "worker", Command: "sidekiq"},
		}
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("returns all entries when names is empty", func(t *testing.T) {
		got, err := FilterEntries(entries, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if diff := cmp.Diff(entries, got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("returns error for unknown names", func(t *testing.T) {
		_, err := FilterEntries(entries, []string{"web", "nonexistent"})
		if err == nil {
			t.Error("expected error for unknown service")
		}
	})
}

func TestParseProcfile(t *testing.T) {
	writeProcfile := func(t *testing.T, content string) string {
		path := filepath.Join(t.TempDir(), "Procfile")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		return path
	}

	t.Run("parses entries skipping comments, blanks, and handles colons in commands", func(t *testing.T) {
		path := writeProcfile(t, "# comment\nweb: bundle exec rails server\n\napi: http://localhost:3000\n")
		entries, err := ParseProcfile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []Entry{
			{Name: "web", Command: "bundle exec rails server"},
			{Name: "api", Command: "http://localhost:3000"},
		}
		if diff := cmp.Diff(expected, entries); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("errors on invalid file", func(t *testing.T) {
		if _, err := ParseProcfile("/nonexistent/path/Procfile"); err == nil {
			t.Error("expected error for missing file")
		}
		if _, err := ParseProcfile(writeProcfile(t, "")); err == nil {
			t.Error("expected error for empty procfile")
		}
		if _, err := ParseProcfile(writeProcfile(t, "web echo hello\n")); err == nil {
			t.Error("expected error for missing colon separator")
		}
	})
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
