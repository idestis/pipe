package ui

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// CursorRow queries the terminal for the current cursor row using DSR
// (Device Status Report). Returns the 1-based row number or an error
// if the terminal doesn't support DSR or stdin is not a TTY.
func CursorRow() (int, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("open /dev/tty: %w", err)
	}
	defer tty.Close() //nolint:errcheck

	fd := int(tty.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, fmt.Errorf("set raw mode: %w", err)
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	// Send DSR query: "\033[6n" → terminal responds "\033[<row>;<col>R"
	if _, err := tty.WriteString("\033[6n"); err != nil {
		return 0, fmt.Errorf("write DSR: %w", err)
	}

	// Read response with timeout
	buf := make([]byte, 32)
	n := 0
	deadline := time.Now().Add(500 * time.Millisecond)

	for n < len(buf) && time.Now().Before(deadline) {
		_ = tty.SetReadDeadline(deadline)
		nr, err := tty.Read(buf[n:])
		if nr > 0 {
			n += nr
			// Check if we got the full response (ends with 'R')
			if buf[n-1] == 'R' {
				break
			}
		}
		if err != nil {
			break
		}
	}

	// Parse "\033[<row>;<col>R"
	var row, col int
	if _, err := fmt.Sscanf(string(buf[:n]), "\033[%d;%dR", &row, &col); err != nil {
		return 0, fmt.Errorf("parse DSR response: %w (got %q)", err, string(buf[:n]))
	}

	return row, nil
}

// TermHeight returns the height (rows) of the terminal, or 0 on error.
func TermHeight() int {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return 0
	}
	defer tty.Close() //nolint:errcheck

	_, h, err := term.GetSize(int(tty.Fd()))
	if err != nil {
		return 0
	}
	return h
}
