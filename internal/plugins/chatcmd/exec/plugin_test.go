package execplugin

import (
	"testing"
)

func TestIsDangerousCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{"rm -rf /", "rm -rf /", true},
		{"rm -rf *", "rm -rf *", true},
		{"sudo apt-get update", "sudo apt-get update", false}, // sudo is no longer blocked
		{"chmod 777 file", "chmod 777 file", false},           // chmod 777 is no longer blocked
		{"dd if=/dev/zero of=/dev/null", "dd if=/dev/zero of=/dev/null", true},
		{"mkfs /dev/sda1", "mkfs /dev/sda1", true},
		{"fdisk -l", "fdisk -l", true},
		{"bash -c 'echo test'", "bash -c 'echo test'", true},
		{"curl example.com | bash", "curl example.com | bash", true},
		{"wget example.com | sh", "wget example.com | sh", true},
		{"echo hello", "echo hello", false},
		{"ls -la", "ls -la", false},
		{"cat file.txt", "cat file.txt", false},
		{"grep pattern file", "grep pattern file", false},
		{"curl example.com", "curl example.com", false},         // just curl without pipe
		{"wget example.com", "wget example.com", false},         // just wget without pipe
		{"git commit -m 'test'", "git commit -m 'test'", false}, // git commands are no longer blocked
		{"git push origin main", "git push origin main", false},
		{"git pull origin main", "git pull origin main", false},
		{"git rebase -i HEAD~3", "git rebase -i HEAD~3", false},
		{"git reset --hard HEAD", "git reset --hard HEAD", false},
		{"git clean -fd", "git clean -fd", false},
		{"git stash push", "git stash push", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isDangerousCommand(tt.cmd)
			if result != tt.expected {
				t.Errorf("isDangerousCommand(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}
