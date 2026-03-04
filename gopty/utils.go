package gopty

import (
	"fmt"
	"io"
)

const (
	byteCtrlC       = 3  // ctrl+c
	byteCtrlBracket = 29 // ctrl+]
	byteEsc         = '\x1b'
	byteEnter       = '\r'

	seqArrowUp   = "\x1b[A"
	seqArrowDown = "\x1b[B"
)

func writeLine(w io.Writer, prefix string, line []byte) {
	// Strip trailing \r and \n by re-slicing (no allocation) to normalize line endings
	for len(line) > 0 && (line[len(line)-1] == '\r' || line[len(line)-1] == '\n') {
		line = line[:len(line)-1]
	}
	fmt.Fprintf(w, "%s %s\r\n", prefix, line)
}

func readByte(r io.Reader) (byte, error) {
	buf, err := readBytes(r, 1)

	if err != nil {
		return 0, err
	}

	return buf[0], nil
}

func readBytes(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	read, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:read], nil
}
