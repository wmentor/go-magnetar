package terminal

import (
	"os"

	"golang.org/x/term"
)

func IsStdinPipe() bool {
	fd := int(os.Stdin.Fd())

	if term.IsTerminal(fd) {
		return false
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return false
	}

	return true
}
