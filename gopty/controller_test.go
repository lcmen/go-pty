package gopty

import (
	"bytes"
	"io"
	"testing"
)

func TestController_Run(t *testing.T) {
	stubManager := func() *Manager {
		return NewManager([]Entry{{Name: "web", Command: "cmd"}}, io.Discard)
	}

	t.Run("ctrl+c shuts down manager", func(t *testing.T) {
		m := stubManager()
		in := bytes.NewReader([]byte{byteCtrlC})
		c := NewController(m, in, io.Discard)

		c.Run()

		if c.err != io.EOF {
			t.Errorf("expected err to be io.EOF, got %v", c.err)
		}
	})

	t.Run("ctrl+] opens dialog then esc returns to normal", func(t *testing.T) {
		m := stubManager()
		// ctrl+], then esc (dialog cancels), then ctrl+c (shutdown)
		in := bytes.NewReader([]byte{byteCtrlBracket, byteEsc, byteCtrlC})
		var out bytes.Buffer
		c := NewController(m, in, &out)

		c.Run()

		if m.Attached() != nil {
			t.Error("expected no process attached after esc")
		}
	})

	t.Run("ctrl+] opens dialog then enter attaches process", func(t *testing.T) {
		m := stubManager()
		// ctrl+] opens dialog, enter selects first, then ctrl+] detaches, then ctrl+c
		in := bytes.NewReader([]byte{byteCtrlBracket, byteEnter, byteCtrlBracket, byteCtrlC})
		c := NewController(m, in, io.Discard)

		c.Run()

		if m.Attached() != nil {
			t.Error("expected no process attached after detach")
		}
	})
}
