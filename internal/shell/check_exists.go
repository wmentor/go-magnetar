package shell

import (
	"os/exec"
	"strings"
)

// CheckExists returns true if the given command exists on the current platform.
// It uses exec.LookPath to check if the command is available in PATH.
func CheckExists(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return false
	}

	// Check if command exists in PATH
	_, err := exec.LookPath(trimmed)
	return err == nil
}
