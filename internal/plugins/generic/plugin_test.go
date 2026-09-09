package genericplugin

import (
	"os"
	"testing"
)

func TestProcessEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single env var",
			input:    "Hello {{env:USER}}",
			expected: "Hello " + os.Getenv("USER"),
		},
		{
			name:     "multiple env vars",
			input:    "User: {{env:USER}}, Home: {{env:HOME}}",
			expected: "User: " + os.Getenv("USER") + ", Home: " + os.Getenv("HOME"),
		},
		{
			name:     "undefined env var",
			input:    "Value: {{env:NONEXISTENT_VAR_12345}}",
			expected: "Value: ",
		},
		{
			name:     "env var at end",
			input:    "Path is {{env:PATH}}",
			expected: "Path is " + os.Getenv("PATH"),
		},
		{
			name:     "env var at start",
			input:    "{{env:HOME}} is home",
			expected: os.Getenv("HOME") + " is home",
		},
		{
			name:     "no env vars",
			input:    "No placeholders here",
			expected: "No placeholders here",
		},
		{
			name:     "malformed placeholder",
			input:    "Hello {{env:USER",
			expected: "Hello {{env:USER",
		},
		{
			name:     "empty env var name",
			input:    "Value: {{env:}}",
			expected: "Value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("ProcessEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
