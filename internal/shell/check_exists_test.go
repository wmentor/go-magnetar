package shell

import (
	"testing"
)

func TestCheckExists(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"non-existent command", "nonexistentcommand12345", false},
		{"ls on Unix-like", "ls", true},
		{"echo", "echo", true},
		{"pwd", "pwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the helper function for tests that expect the command to exist
			if tt.expected {
				if !CheckCommandExists(t, tt.cmd) {
					return // Test was skipped by CheckCommandExists
				}
			}
			result := CheckExists(tt.cmd)
			if result != tt.expected {
				t.Errorf("CheckExists(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}
