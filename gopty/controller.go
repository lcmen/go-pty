package gopty

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

type Controller struct {
	err     atomic.Pointer[error]
	manager *Manager
	stdin   io.Reader
	stdout  io.Writer
}

func NewController(manager *Manager, stdin io.Reader, stdout io.Writer) *Controller {
	return &Controller{
		manager: manager,
		stdin:   stdin,
		stdout:  stdout,
	}
}

func (c *Controller) Run() {
	for c.err.Load() == nil {
		if c.manager.Attached() != nil {
			c.handleAttached()
		} else {
			c.handleAllOut()
		}
	}
}

func (c *Controller) Shutdown() {
	eof := io.EOF
	c.err.Store(&eof)
	c.manager.Shutdown()
}

func (c *Controller) Cleanup() {
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		c.stdin.Read(buf)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
	}
}

func (c *Controller) handleAllOut() {
	buf, err := readByte(c.stdin)
	if err != nil {
		c.err.Store(&err)
		return
	}

	switch buf {
	case byteCtrlBracket:
		d := NewDialog(c.manager.Processes(), c.stdin, c.stdout)
		idx, ok := d.Open()
		if ok {
			c.manager.Attach(idx)
			p := c.manager.Attached()
			fmt.Fprintf(c.stdout, "\r\n[go-pty] Attached to %s (ctrl+] to detach)\r\n", p.Name)
		}
	case byteCtrlC:
		fmt.Fprintf(c.stdout, "[go-pty] Terminating...\r\n")
		c.Shutdown()
	}
}

func (c *Controller) handleAttached() {
	buf, err := readByte(c.stdin)
	if err != nil {
		c.err.Store(&err)
		return
	}

	switch buf {
	case byteCtrlBracket:
		name := c.manager.Attached().Name
		c.manager.Detach()
		fmt.Fprintf(c.stdout, "\r\n[go-pty] Detached from %s\r\n", name)
	case byteCtrlC:
		fmt.Fprintf(c.stdout, "\r\n[go-pty] (press ctrl+] to detach first, then press ctrl+c again)\r\n")
	default:
		c.manager.WriteToAttached([]byte{buf})
	}
}
