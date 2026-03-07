package gopty

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"
)

func TestController_Run(t *testing.T) {
	stubManager := func() *Manager {
		return NewManager([]Entry{{Name: "web", Command: "cmd"}}, io.Discard)
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
		if m.Attached() != nil {
			t.Error("expected no process attached after esc")
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
		if m.Attached() != nil {
			t.Error("expected no process attached after detach")
		}
	})

	t.Run("keystrokes are forwarded to attached process", func(t *testing.T) {
		r, w, _ := os.Pipe()
		m := stubManager()
		m.processes[0].pty = w
		m.Attach(0)
		// Type 'a', 'b', ctrl+c (ignored in attached mode), then ctrl+] to detach, ctrl+c to shutdown
		c := NewController(m, stubKeypresses('a', 'b', byteCtrlC, byteCtrlBracket, byteCtrlC), io.Discard)

		c.Run()

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		// Expect '\n' from Attach(), then 'ab' (ctrl+c is ignored)
		if diff := strings.Compare("\nab", buf.String()); diff != 0 {
			t.Errorf("forwarded data mismatch: expected %q, got %q", "\nab", buf.String())
		}
	})
}
