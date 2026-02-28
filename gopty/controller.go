package gopty

import "io"

type Controller struct {
	err     error
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
	for c.err == nil {
		if c.manager.Attached() != nil {
			c.handleAttached()
		} else {
			c.handleAllOut()
		}
	}
}

func (c *Controller) Shutdown() {
	c.err = io.EOF
	c.manager.Shutdown()
}

func (c *Controller) handleAllOut() {
	buf, err := readBytes(c.stdin, 1)
	if err != nil {
		c.err = err
		return
	}

	switch buf[0] {
	case byteCtrlC:
		c.Shutdown()
	case byteCtrlBracket:
		d := NewDialog(c.manager.Processes(), c.stdin, c.stdout)
		idx, ok := d.Open()
		if ok {
			c.manager.Attach(idx)
		}
	}
}

func (c *Controller) handleAttached() {
	buf, err := readBytes(c.stdin, 1)
	if err != nil {
		c.err = err
		return
	}

	if buf[0] == byteCtrlBracket {
		c.manager.Detach()
	}

	// Other bytes ignored for now (Phase 5b: forward to process)
}
