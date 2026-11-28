package shell

import (
	"fmt"
	"testing"
)

func TestGrepCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"grep magnetar -r .", true},
		{"grep magnetar -r ..", true},
		{"ls .", true},
		{"cat ./file.txt", true},
	}

	for _, tt := range tests {
		// Skip tests for commands that don't exist on this platform
		if tt.expected && !CheckCommandExists(t, tt.input) {
			continue
		}
		result := CheckCommand(tt.input)
		status := "✓"
		if result != tt.expected {
			status = "✗"
		}
		fmt.Printf("%s '%s' -> got=%v, expected=%v\n", status, tt.input, result, tt.expected)
		if result != tt.expected {
			t.Errorf("CheckCommand(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
