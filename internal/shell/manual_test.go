package shell

import (
	"fmt"
	"testing"
)

func TestManualCheck(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"time `12:00`", false},       // Invalid command in backticks
		{"time `date`", true},         // Valid command in backticks
		{"time date", true},           // Valid command
		{"time ls", true},             // Valid command
		{"time where is grep", false}, // English text
		{"grep pattern file", true},   // Valid command
		{"ls -la", true},              // Valid command with flag
		{"good morning", false},       // English text
	}

	for _, tt := range tests {
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
