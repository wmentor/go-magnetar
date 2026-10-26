package execplugin

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/command/interceptor"
)

func TestInterceptor_IsCommandBlocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{"rm -rf /", "rm -rf /", true},
		{"rm -rf *", "rm -rf *", true},
		{"sudo apt-get update", "sudo apt-get update", true}, // sudo is now blocked
		{"chmod 777 file", "chmod 777 file", true},           // chmod is now blocked
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
		{"curl example.com", "curl example.com", false},        // just curl without pipe
		{"wget example.com", "wget example.com", false},        // just wget without pipe
		{"git commit -m 'test'", "git commit -m 'test'", true}, // git commands are now blocked
		{"git push origin main", "git push origin main", true},
		{"git pull origin main", "git pull origin main", true},
		{"git rebase -i HEAD~3", "git rebase -i HEAD~3", true},
		{"git reset --hard HEAD", "git reset --hard HEAD", true},
		{"git clean -fd", "git clean -fd", true},
		{"git stash push", "git stash push", true},
		{"ls; rm -rf /tmp", "ls; rm -rf /tmp", true},                         // rm after semicolon should be blocked
		{"test && sudo apt update", "test && sudo apt update", true},         // sudo after && should be blocked
		{"ls | grep test; rm -rf /tmp", "ls | grep test; rm -rf /tmp", true}, // rm after pipe
		{"cd /tmp && chmod 777 file", "cd /tmp && chmod 777 file", true},     // chmod after && should be blocked
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			i := interceptor.New()
			result := i.IsBlocked(tt.cmd)
			if result != tt.expected {
				t.Errorf("IsBlocked(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}
