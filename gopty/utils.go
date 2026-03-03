package gopty

import (
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
