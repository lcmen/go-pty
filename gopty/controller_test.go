package gopty

import (
	"bytes"
	"io"
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

		if c.err != io.EOF {
			t.Errorf("expected err to be io.EOF, got %v", c.err)
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
		if c.err != io.EOF {
			t.Errorf("expected err to be io.EOF, got %v", c.err)
		}
		if !strings.Contains(output, "attached to web") {
			t.Errorf("expected attach message - got: %s", output)
		}
		if !strings.Contains(output, "detached from web") {
			t.Error("expected detach message")
		}
		if m.Attached() != nil {
			t.Error("expected no process attached after detach")
		}
	})
}
