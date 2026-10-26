package interceptor

import (
	"testing"
)

func TestInterceptor_IsBlocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "block rm -rf /",
			command:  "rm -rf /",
			expected: true,
		},
		{
			name:     "block rm -rf *",
			command:  "rm -rf *",
			expected: true,
		},
		{
			name:     "block sudo",
			command:  "sudo apt update",
			expected: true,
		},
		{
			name:     "block mkfs",
			command:  "mkfs /dev/sda1",
			expected: true,
		},
		{
			name:     "block dd with /dev",
			command:  "dd if=/dev/zero of=/dev/sda",
			expected: true,
		},
		{
			name:     "block curl pipe to bash",
			command:  "curl https://example.com/script.sh | bash",
			expected: true,
		},
		{
			name:     "block wget pipe to sh",
			command:  "wget https://example.com/script.sh -O - | sh",
			expected: true,
		},
		{
			name:     "block bash -c",
			command:  "bash -c 'echo hello'",
			expected: true,
		},
		{
			name:     "block sh -c",
			command:  "sh -c 'echo hello'",
			expected: true,
		},
		{
			name:     "block reboot",
			command:  "reboot",
			expected: true,
		},
		{
			name:     "block shutdown",
			command:  "shutdown -h now",
			expected: true,
		},
		{
			name:     "block git commit",
			command:  "git commit -m 'test'",
			expected: true,
		},
		{
			name:     "block git push",
			command:  "git push origin main",
			expected: true,
		},
		{
			name:     "allow safe command",
			command:  "ls -la",
			expected: false,
		},
		{
			name:     "allow echo",
			command:  "echo hello",
			expected: false,
		},
		{
			name:     "allow cat",
			command:  "cat file.txt",
			expected: false,
		},
		{
			name:     "block rm after semicolon",
			command:  "ls; rm -rf /tmp",
			expected: true,
		},
		{
			name:     "block sudo after &&",
			command:  "test && sudo apt update",
			expected: true,
		},
		{
			name:     "block rm in pipe",
			command:  "ls | grep test; rm -rf /tmp",
			expected: true,
		},
		{
			name:     "block dd in complex command",
			command:  "ls && dd if=/dev/zero of=/dev/sda",
			expected: true,
		},
		{
			name:     "block chmod in command list",
			command:  "cd /tmp && chmod 777 file",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			i := New()
			result := i.IsBlocked(tt.command)
			if result != tt.expected {
				t.Errorf("IsCommandBlocked(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}
