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
		{
			name:  "single file",
			input: "content: $file:test.txt",
			setup: func() {
				os.WriteFile("test.txt", []byte("file content"), 0644)
			},
			cleanup: func() {
				os.Remove("test.txt")
			},
			expected: "content: file content",
		},
		{
			name:  "multiple files",
			input: "$file:a.txt and $file:b.txt",
			setup: func() {
				os.WriteFile("a.txt", []byte("first"), 0644)
				os.WriteFile("b.txt", []byte("second"), 0644)
			},
			cleanup: func() {
				os.Remove("a.txt")
				os.Remove("b.txt")
			},
			expected: "first and second",
		},
		{
			name:     "missing file returns empty",
			input:    "value: $file:missing.txt",
			expected: "value: ",
		},
		{
			name:  "env in file path",
			input: "$file:$env:FILE_VAR",
			setup: func() {
				os.Setenv("FILE_VAR", "msg.txt")
				os.WriteFile("msg.txt", []byte("Hello"), 0644)
			},
			cleanup: func() {
				os.Remove("msg.txt")
				os.Unsetenv("FILE_VAR")
			},
			expected: "Hello",
		},
		{
			name:  "file and env mixed",
			input: "$file:msg.txt $env:USER",
			setup: func() {
				os.WriteFile("msg.txt", []byte("Hello"), 0644)
				os.Setenv("USER", "World")
			},
			cleanup: func() {
				os.Remove("msg.txt")
				os.Unsetenv("USER")
			},
			expected: "Hello World",
		},
		{
			name:  "file with newlines",
			input: "$file:multiline.txt",
			setup: func() {
				os.WriteFile("multiline.txt", []byte("line1\nline2\nline3"), 0644)
			},
			cleanup: func() {
				os.Remove("multiline.txt")
			},
			expected: "line1\nline2\nline3",
		},
		{
			name:  "file with path",
			input: "$file:subdir/test.txt",
			setup: func() {
				os.Mkdir("subdir", 0755)
				os.WriteFile("subdir/test.txt", []byte("nested"), 0644)
			},
			cleanup: func() {
				os.Remove("subdir/test.txt")
				os.Remove("subdir")
			},
			expected: "nested",
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
