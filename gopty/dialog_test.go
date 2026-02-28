package gopty

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func stubProcesses() []*Process {
	return []*Process{
		{Entry: Entry{Name: "web"}, Color: "\033[31m"},
		{Entry: Entry{Name: "worker"}, Color: "\033[32m"},
		{Entry: Entry{Name: "css"}, Color: "\033[33m"},
	}
}

func TestDialog_Open(t *testing.T) {
	t.Run("enter selects first item by default then leaves alternate screen", func(t *testing.T) {
		in := bytes.NewReader([]byte{byteEnter})
		var out bytes.Buffer

		idx, ok := NewDialog(stubProcesses(), in, &out).Open()
		output := out.String()

		if diff := cmp.Diff(true, ok); diff != "" {
			t.Errorf("ok mismatch (-expected +got):\n%s", diff)
		}
		if diff := cmp.Diff(0, idx); diff != "" {
			t.Errorf("index mismatch (-expected +got):\n%s", diff)
		}
		if !strings.Contains(output, "1. web") {
			t.Error("expected output to contain '1. web'")
		}
		if !strings.Contains(output, "2. worker") {
			t.Error("expected output to contain '2. worker'")
		}
		if !strings.Contains(output, "3. css") {
			t.Error("expected output to contain '3. css'")
		}
		if !strings.Contains(output, enterAltScreen) {
			t.Error("expected output to contain enterAltScreen")
		}
		if !strings.Contains(output, leaveAltScreen) {
			t.Error("expected output to contain leaveAltScreen")
		}
	})

	t.Run("arrow keys navigate and clamp within bounds", func(t *testing.T) {
		// Up (clamps at 0), down x4 (clamps at 2), up (back to 1), enter
		in := bytes.NewReader([]byte(seqArrowUp + seqArrowDown + seqArrowDown + seqArrowDown + seqArrowDown + seqArrowUp + string(byteEnter)))
		var out bytes.Buffer

		idx, ok := NewDialog(stubProcesses(), in, &out).Open()

		if diff := cmp.Diff(true, ok); diff != "" {
			t.Errorf("ok mismatch (-expected +got):\n%s", diff)
		}
		if diff := cmp.Diff(1, idx); diff != "" {
			t.Errorf("index mismatch (-expected +got):\n%s", diff)
		}
	})

	t.Run("esc cancels", func(t *testing.T) {
		in := bytes.NewReader([]byte{byteEsc})
		var out bytes.Buffer

		_, ok := NewDialog(stubProcesses(), in, &out).Open()

		if diff := cmp.Diff(false, ok); diff != "" {
			t.Errorf("ok mismatch (-expected +got):\n%s", diff)
		}
	})
}
