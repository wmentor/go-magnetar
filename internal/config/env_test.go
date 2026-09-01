package config

import (
	"os"
	"testing"
)

func TestResolveEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		setup    func()
		cleanup  func()
		expected string
	}{
		{
			name:     "no env vars",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:  "single env var",
			input: "key is $env:TEST_KEY",
			setup: func() {
				os.Setenv("TEST_KEY", "secret123")
			},
			cleanup: func() {
				os.Unsetenv("TEST_KEY")
			},
			expected: "key is secret123",
		},
		{
			name:  "multiple env vars",
			input: "$env:KEY1 and $env:KEY2",
			setup: func() {
				os.Setenv("KEY1", "value1")
				os.Setenv("KEY2", "value2")
			},
			cleanup: func() {
				os.Unsetenv("KEY1")
				os.Unsetenv("KEY2")
			},
			expected: "value1 and value2",
		},
		{
			name:     "missing env var returns empty",
			input:    "value: $env:MISSING_VAR",
			expected: "value: ",
		},
		{
			name:  "mixed existing and missing",
			input: "$env:EXISTS/$env:MISSING",
			setup: func() {
				os.Setenv("EXISTS", "present")
			},
			cleanup: func() {
				os.Unsetenv("EXISTS")
			},
			expected: "present/",
		},
		{
			name:  "env var at start",
			input: "$env:VAR1 is good",
			setup: func() {
				os.Setenv("VAR1", "apple")
			},
			cleanup: func() {
				os.Unsetenv("VAR1")
			},
			expected: "apple is good",
		},
		{
			name:  "env var at end",
			input: "value is $env:VAR2",
			setup: func() {
				os.Setenv("VAR2", "banana")
			},
			cleanup: func() {
				os.Unsetenv("VAR2")
			},
			expected: "value is banana",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			result := ResolveEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
