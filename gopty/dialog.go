package gopty

import (
	"fmt"
	"io"
)

const (
	enterAltScreen = "\033[?1049h"
	leaveAltScreen = "\033[?1049l"
	showCursor     = "\033[?25h"
	hideCursor     = "\033[?25l"
	clearScreen    = "\033[2J"
	cursorHome     = "\033[H"
	reverseVideo   = "\033[7m"
)

type key int

const (
	keyUp key = iota
	keyDown
	keyEnter
	keyEsc
	keyOther
)

type Dialog struct {
	in        io.Reader
	out       io.Writer
	processes []*Process
	selected  int
}

func NewDialog(processes []*Process, in io.Reader, out io.Writer) *Dialog {
	return &Dialog{in: in, out: out, processes: processes}
}

func (d *Dialog) Open() (int, bool) {
	fmt.Fprint(d.out, enterAltScreen)
	d.render()

	for {
		k, err := d.readKey()
		if err != nil {
			d.close()
			return 0, false
		}

		switch k {
		case keyEnter:
			d.close()
			return d.selected, true
		case keyEsc:
			d.close()
			return 0, false
		case keyUp:
			if d.selected > 0 {
				d.selected--
				d.render()
			}
		case keyDown:
			if d.selected < len(d.processes)-1 {
				d.selected++
				d.render()
			}
		}
	}
}

func (d *Dialog) close() {
	fmt.Fprint(d.out, showCursor+leaveAltScreen)
}

func (d *Dialog) render() {
	fmt.Fprint(d.out, clearScreen+cursorHome+hideCursor)
	fmt.Fprint(d.out, "Select a process (↑/↓ navigate, Enter select, Esc cancel):\r\n\r\n")

	for i, p := range d.processes {
		if i == d.selected {
			fmt.Fprintf(d.out, "  %s> %d. %s\033[0m\r\n", reverseVideo, i+1, p.Name)
		} else {
			fmt.Fprintf(d.out, "  %s  %d. %s\033[0m\r\n", p.Color, i+1, p.Name)
		}
	}
}

func (d *Dialog) readKey() (key, error) {
	buf, err := readBytes(d.in, 3)
	if err != nil {
		return keyOther, err
	}

	switch string(buf) {
	case seqArrowUp:
		return keyUp, nil
	case seqArrowDown:
		return keyDown, nil
	case seqEnter:
		return keyEnter, nil
	case seqEsc:
		return keyEsc, nil
	default:
		return keyOther, nil
	}
}
