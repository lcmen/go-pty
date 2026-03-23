package gopty

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
)

type Controller struct {
	err       atomic.Pointer[error]
	mgr       atomic.Pointer[Manager]
	mode      OutputMode
	gen       atomic.Int64
	restarted chan struct{}
	stdin     io.Reader
	stdout    io.Writer
}

func NewController(manager *Manager, stdin io.Reader, stdout io.Writer) *Controller {
	c := &Controller{
		mode:      OutputAll,
		restarted: make(chan struct{}, 1),
		stdin:     stdin,
		stdout:    stdout,
	}
	c.mgr.Store(manager)
	return c
}

func (c *Controller) ResizeAll(ws *pty.Winsize) {
	c.manager().ResizeAll(ws)
}

func (c *Controller) Run() {
	for c.err.Load() == nil {
		switch c.mode {
		case OutputAttached:
			c.handleAttached()
		default:
			c.handleAllOut()
		}
	}
}

func (c *Controller) Wait() {
	idx := int64(0)
	for {
		c.manager().WaitAll()

		// Manager crashed (no restart pending), signal inputLoop to exit
		if idx == c.gen.Load() {
			c.err.Store(&io.EOF)
			break
		}

		// idx < gen, wait for restart signal
		<-c.restarted
		idx++
	}
	c.cleanup()
}

func (c *Controller) Shutdown() {
	eof := io.EOF
	c.err.Store(&eof)
	c.manager().Shutdown()
}

func (c *Controller) cleanup() {
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
		d := NewDialog(c.manager().Processes(), c.stdin, c.stdout)
		idx, ok := d.Open()
		if ok {
			p, err := c.manager().Attach(idx)
			if err == nil {
				c.mode = OutputAttached
				fmt.Fprintf(c.stdout, "\r\n[go-pty] Attached to %s (ctrl+] to detach)\r\n", p.Name)
			}
		}
	case byteCtrlR:
		fmt.Fprintf(c.stdout, "[go-pty] Restarting all processes...\r\n")
		c.gen.Add(1)
		newManager, err := c.manager().Restart()
		if err != nil {
			fmt.Fprintf(c.stdout, "[go-pty] Restart failed: %v\r\n", err)
			close(c.restarted)
			c.Shutdown()
			return
		}
		c.mgr.Store(newManager)
		c.restarted <- struct{}{}
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
		p := c.manager().Detach()
		c.mode = OutputAll
		fmt.Fprintf(c.stdout, "\r\n[go-pty] Detached from %s\r\n", p.Name)
	case byteCtrlC:
		fmt.Fprintf(c.stdout, "\r\n[go-pty] (press ctrl+] to detach first, then press ctrl+c again)\r\n")
	default:
		if _, err := c.manager().WriteToAttached([]byte{buf}); err != nil {
			c.err.Store(&err)
		}
	}
}

func (c *Controller) manager() *Manager {
	return c.mgr.Load()
}
