package gopty

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/google/go-cmp/cmp"
)

func TestController_Run(t *testing.T) {
	stubManager := func() *Manager {
		return NewManager([]Entry{{Name: "web", Command: "sleep 60"}}, io.Discard, nil)
	}

	stubKeypresses := func(keys ...byte) io.Reader {
		return iotest.OneByteReader(bytes.NewReader(keys))
	}

	t.Run("ctrl+] opens dialog then esc returns to normal", func(t *testing.T) {
		var out bytes.Buffer
		m := stubManager()
		c := NewController(m, stubKeypresses(byteCtrlBracket, byteEsc, byteCtrlC), &out)

		c.Run()

		if c.err.Load() == nil {
			t.Error("expected controller to be stopped")
		}
		if c.mode != OutputAll {
			t.Error("expected controller to be in all-out mode")
		}
	})

	t.Run("ctrl+] opens dialog then enter attaches process", func(t *testing.T) {
		var out bytes.Buffer
		m := stubManager()
		c := NewController(m, stubKeypresses(byteCtrlBracket, byteEnter, byteCtrlBracket, byteCtrlC), &out)

		c.Run()

		output := out.String()
		if c.err.Load() == nil {
			t.Error("expected controller to be stopped")
		}
		if !strings.Contains(output, "Attached to web") {
			t.Errorf("expected attach message - got: %s", output)
		}
		if !strings.Contains(output, "Detached from web") {
			t.Error("expected detach message")
		}
		if c.mode != OutputAll {
			t.Error("expected controller to be in all-out mode after detach")
		}
	})

	t.Run("ctrl+r restarts all processes", func(t *testing.T) {
		var out bytes.Buffer
		m := NewManager([]Entry{{Name: "web", Command: "sleep 60"}}, io.Discard, nil)
		if err := m.StartAll(); err != nil {
			t.Fatalf("StartAll failed: %v", err)
		}
		oldPID := m.processes[0].cmd.Process.Pid

		c := NewController(m, stubKeypresses(byteCtrlR, byteCtrlC), &out)
		go c.Run()
		c.Wait()

		output := out.String()
		if !strings.Contains(output, "Restarting all processes") {
			t.Errorf("expected restart message, got: %s", output)
		}
		newPID := c.manager().processes[0].cmd.Process.Pid
		if newPID == oldPID {
			t.Errorf("expected new PID after restart, got same PID %d", oldPID)
		}
	})

	t.Run("keystrokes are forwarded to attached process", func(t *testing.T) {
		r, w, _ := os.Pipe()
		m := stubManager()
		m.processes[0].pty = w
		m.Attach(0)
		// Type 'a', 'b', ctrl+c (ignored in attached mode), then ctrl+] to detach, ctrl+c to shutdown
		c := NewController(m, stubKeypresses('a', 'b', byteCtrlC, byteCtrlBracket, byteCtrlC), io.Discard)
		c.mode = OutputAttached

		c.Run()

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		// Expect '\n' from Attach(), then 'ab' (ctrl+c is ignored)
		if diff := cmp.Diff("\nab", buf.String()); diff != "" {
			t.Errorf("forwarded data mismatch (-expected +got):\n%s", diff)
		}
	})
}
