package gopty

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFile_Parse(t *testing.T) {
	t.Run("parses simple name:command lines", func(t *testing.T) {
		f := &File{lines: []string{
			"# This is a comment",
			"web: bundle exec rails server",
			"api: http://localhost:3000",
			"",
			"worker: bundle exec sidekiq",
		}}
		entries, err := f.Parse()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []Entry{
			{Name: "web", Command: "bundle exec rails server"},
			{Name: "api", Command: "http://localhost:3000"},
			{Name: "worker", Command: "bundle exec sidekiq"},
		}
		if diff := cmp.Diff(expected, entries); diff != "" {
			t.Errorf("Parse() mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("errors on invalid syntax", func(t *testing.T) {
		f := &File{lines: []string{
			"web echo hello",
		}}
		_, err := f.Parse()

		if err == nil {
			t.Error("expected error for line without colon separator, got nil")
		}
	})

	t.Run("errors on empty content", func(t *testing.T) {
		f := &File{lines: []string{}}
		_, err := f.Parse()

		if err == nil {
			t.Error("expected error for empty content, got nil")
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("reads file and stores lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "Procfile")
		content := "web: echo hello\nworker: echo world\n"

		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		pf, err := Open(tmpFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"web: echo hello", "worker: echo world"}
		if diff := cmp.Diff(expected, pf.lines); diff != "" {
			t.Errorf("File.lines mismatch (-expected +got):\n%s", diff)
		}
		if pf.path != tmpFile {
			t.Errorf("File.path = %q, expected %q", pf.path, tmpFile)
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := Open("/nonexistent/path/Procfile")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}
