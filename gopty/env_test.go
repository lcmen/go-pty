package gopty

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewEnv(t *testing.T) {
	t.Run("trims key and value", func(t *testing.T) {
		e := NewEnv("  FOO  ", "  bar  ")
		if diff := cmp.Diff(Env{Key: "FOO", Value: "bar"}, e); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})
}

func TestEnv_Environ(t *testing.T) {
	e := Env{Key: "FOO", Value: "bar"}
	if diff := cmp.Diff("FOO=bar", e.Environ()); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}
}

func TestEnv_Expand(t *testing.T) {
	t.Run("returns value unchanged when no env vars", func(t *testing.T) {
		e := Env{Key: "FOO", Value: "bar"}
		got := e.Expand(map[string]string{})
		if diff := cmp.Diff("bar", got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("expands env var from expanded map", func(t *testing.T) {
		e := Env{Key: "LOG", Value: "${BASE}/logs"}
		expanded := map[string]string{"BASE": "/app"}
		got := e.Expand(expanded)
		if diff := cmp.Diff("/app/logs", got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("falls back to os environment", func(t *testing.T) {
		e := Env{Key: "HOME", Value: "${HOME}"}
		got := e.Expand(map[string]string{})
		if got == "" {
			t.Error("expected non-empty result from os env fallback")
		}
	})

	t.Run("handles multiple references", func(t *testing.T) {
		e := Env{Key: "PATH", Value: "${FOO}:${BAR}"}
		expanded := map[string]string{"FOO": "/usr", "BAR": "/bin"}
		got := e.Expand(expanded)
		if diff := cmp.Diff("/usr:/bin", got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("returns empty when var not found and not in os env", func(t *testing.T) {
		e := Env{Key: "UNKNOWN", Value: "${NONEXISTENT_VAR_12345}"}
		got := e.Expand(map[string]string{})
		if diff := cmp.Diff("", got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})
}

func TestExpandAll(t *testing.T) {
	t.Run("expands env vars in order", func(t *testing.T) {
		envs := []Env{
			{Key: "BASE", Value: "/app"},
			{Key: "LOG", Value: "${BASE}/logs"},
			{Key: "CONFIG", Value: "${LOG}/config"},
		}
		got := ExpandAll(envs)
		expected := []Env{
			{Key: "BASE", Value: "/app"},
			{Key: "LOG", Value: "/app/logs"},
			{Key: "CONFIG", Value: "/app/logs/config"},
		}
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("handles multiple references", func(t *testing.T) {
		envs := []Env{
			{Key: "FOO", Value: "/usr"},
			{Key: "BAR", Value: "/bin"},
			{Key: "PATH", Value: "${FOO}:${BAR}"},
		}
		got := ExpandAll(envs)
		expected := []Env{
			{Key: "FOO", Value: "/usr"},
			{Key: "BAR", Value: "/bin"},
			{Key: "PATH", Value: "/usr:/bin"},
		}
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Errorf("mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		got := ExpandAll([]Env{})
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})
}
