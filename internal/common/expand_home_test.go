package common_test

import (
	"os/user"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/common"
)

func TestExpandHome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expand tilde in path",
			input:    "~/Downloads/file.txt",
			expected: filepath.Join(userHomeDir(t), "Downloads", "file.txt"),
		},
		{
			name:     "no tilde in path",
			input:    "/home/user/file.txt",
			expected: "/home/user/file.txt",
		},
		{
			name:     "tilde in middle of path",
			input:    "/path~/file.txt",
			expected: "/path~/file.txt",
		},
		{
			name:     "tilde with other prefix",
			input:    "file~/test.txt",
			expected: "file~/test.txt",
		},
		{
			name:     "just tilde",
			input:    "~",
			expected: userHomeDir(t),
		},
		{
			name:     "tilde slash",
			input:    "~/",
			expected: userHomeDir(t) + "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.ExpandHome(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func userHomeDir(t *testing.T) string {
	t.Helper()
	usr, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	return usr.HomeDir
}
