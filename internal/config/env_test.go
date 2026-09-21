package config

import (
	"os"
	"path/filepath"
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

	t.Run("file relative to baseDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content from baseDir"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := ResolveEnvVars("content: $file:test.txt", tmpDir)
		expected := "content: content from baseDir"
		if result != expected {
			t.Errorf("ResolveEnvVars with baseDir = %q, want %q", result, expected)
		}
	})

	t.Run("file with absolute path ignores baseDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "absolute.txt")
		if err := os.WriteFile(testFile, []byte("absolute content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := ResolveEnvVars("$file:"+testFile, "/some/other/dir")
		expected := "absolute content"
		if result != expected {
			t.Errorf("ResolveEnvVars with absolute path = %q, want %q", result, expected)
		}
	})

	t.Run("file with ~/ path", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		tmpDir := t.TempDir()

		testDir := filepath.Join(tmpDir, "testdir")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		testFile := filepath.Join(testDir, "tilde.txt")
		if err := os.WriteFile(testFile, []byte("tilde content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		relPath, _ := filepath.Rel(home, testDir)
		result := ResolveEnvVars("$file:~/" + relPath + "/tilde.txt")
		expected := "tilde content"
		if result != expected {
			t.Errorf("ResolveEnvVars with ~/ path = %q, want %q", result, expected)
		}
	})

	t.Run("file with nested env var", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "nested.txt")
		if err := os.WriteFile(testFile, []byte("nested content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		os.Setenv("FILE_VAR", "nested.txt")
		defer os.Unsetenv("FILE_VAR")

		result := ResolveEnvVars("$file:$env:FILE_VAR", tmpDir)
		expected := "nested content"
		if result != expected {
			t.Errorf("ResolveEnvVars with nested env = %q, want %q", result, expected)
		}
	})

	t.Run("config config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yml")
		configContent := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      api_key: $file:secrets/api_key.txt
`
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		secretsDir := filepath.Join(tmpDir, "secrets")
		if err := os.MkdirAll(secretsDir, 0755); err != nil {
			t.Fatalf("failed to create secrets dir: %v", err)
		}

		apiKeyFile := filepath.Join(secretsDir, "api_key.txt")
		if err := os.WriteFile(apiKeyFile, []byte("my-secret-key"), 0644); err != nil {
			t.Fatalf("failed to create api key file: %v", err)
		}

		cfg, err := Load(configFile)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		apiKey := cfg.ProfileParamString("llm.api_key")
		expected := "my-secret-key"
		if apiKey != expected {
			t.Errorf("ProfileParamString = %q, want %q", apiKey, expected)
		}
	})
}
