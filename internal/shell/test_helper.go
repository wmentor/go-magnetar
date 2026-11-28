package shell

import (
	"os/exec"
	"strings"
	"testing"
)

// CheckCommandExists is a test helper that checks if a command exists and skips the test if it doesn't.
// It calls t.Helper() to ensure the test failure points to the calling test function.
// Returns true if the command exists, false otherwise (and skips the test).
func CheckCommandExists(t *testing.T, cmd string) bool {
	t.Helper()

	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		t.Skipf("test skipped: empty command")
		return false
	}

	// Extract just the command name (first word)
	// This handles cases like "ls -la", "grep pattern file", etc.
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return false
	}
	commandName := parts[0]

	// Check if command exists in PATH
	_, err := exec.LookPath(commandName)
	return err == nil
}
