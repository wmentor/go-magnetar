package shell

import (
	"testing"
)

func TestCheckCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Should NOT match (plain English text)
		{
			name:     "plain english time is good",
			input:    "time is good",
			expected: false,
		},
		{
			name:     "plain english what time is it",
			input:    "what time is it",
			expected: false,
		},
		{
			name:     "plain english where are you",
			input:    "where are you",
			expected: false,
		},
		{
			name:     "plain english hello world",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "plain english good morning",
			input:    "good morning",
			expected: false,
		},
		{
			name:     "plain english how are you",
			input:    "how are you",
			expected: false,
		},
		{
			name:     "plain english what is your name",
			input:    "what is your name",
			expected: false,
		},
		{
			name:     "plain english where is the library",
			input:    "where is the library",
			expected: false,
		},
		{
			name:     "plain english this is a test",
			input:    "this is a test",
			expected: false,
		},
		{
			name:     "plain english what are you doing",
			input:    "what are you doing",
			expected: false,
		},
		{
			name:     "plain english where do you live",
			input:    "where do you live",
			expected: false,
		},
		{
			name:     "plain english why is the sky blue",
			input:    "why is the sky blue",
			expected: false,
		},
		{
			name:     "plain english who is your creator",
			input:    "who is your creator",
			expected: false,
		},
		{
			name:     "plain english it is good",
			input:    "it is good",
			expected: false,
		},
		{
			name:     "plain english this is fine",
			input:    "this is fine",
			expected: false,
		},

		// Should match (real commands)
		{
			name:     "command ls",
			input:    "ls",
			expected: true,
		},
		{
			name:     "command ls with flag",
			input:    "ls -la",
			expected: true,
		},
		{
			name:     "command echo",
			input:    "echo hello",
			expected: true,
		},
		{
			name:     "command cat file",
			input:    "cat file.txt",
			expected: true,
		},
		{
			name:     "command pwd",
			input:    "pwd",
			expected: true,
		},
		{
			name:     "command git status",
			input:    "git status",
			expected: true,
		},
		{
			name:     "command whoami",
			input:    "whoami",
			expected: true,
		},
		{
			name:     "command grep with pattern and file",
			input:    "grep pattern file",
			expected: true,
		},
		{
			name:     "command grep with recursive flag and dot",
			input:    "grep magnetar -r .",
			expected: true,
		},
		{
			name:     "command grep with recursive flag and parent dir",
			input:    "grep magnetar -r ..",
			expected: true,
		},
		{
			name:     "command ls with dot",
			input:    "ls .",
			expected: true,
		},
		{
			name:     "command cat with relative path",
			input:    "cat ./file.txt",
			expected: true,
		},
		{
			name:     "command find with options",
			input:    "find . -name *.go",
			expected: true,
		},
		{
			name:     "command make build",
			input:    "make build",
			expected: true,
		},
		{
			name:     "command go run",
			input:    "go run main.go",
			expected: true,
		},
		{
			name:     "command time (shell builtin)",
			input:    "time",
			expected: true,
		},
		{
			name:     "command time with command",
			input:    "time ls",
			expected: true,
		},
		{
			name:     "command curl with URL",
			input:    "curl google.com",
			expected: true,
		},
		{
			name:     "command python3 with script",
			input:    "python3 script.py",
			expected: true,
		},
		{
			name:     "command node with script",
			input:    "node app.js",
			expected: true,
		},
		{
			name:     "command ssh with hostname",
			input:    "ssh user@host",
			expected: true,
		},

		// Should match (shell syntax)
		{
			name:     "pipe ls and grep",
			input:    "ls | grep go",
			expected: true,
		},
		{
			name:     "semicolon ls and echo",
			input:    "ls; echo done",
			expected: true,
		},
		{
			name:     "and ls and echo",
			input:    "ls && echo done",
			expected: true,
		},
		{
			name:     "or ls or echo",
			input:    "ls || echo failed",
			expected: true,
		},
		{
			name:     "command true",
			input:    "true",
			expected: true,
		},
		{
			name:     "command false",
			input:    "false",
			expected: true,
		},
		{
			name:     "command exit with code",
			input:    "exit 0",
			expected: true,
		},
		{
			name:     "command cd with path",
			input:    "cd /tmp",
			expected: true,
		},
		{
			name:     "command export with variable",
			input:    "export VAR=value",
			expected: true,
		},
		{
			name:     "test command with brackets",
			input:    "[ -f file.txt ]",
			expected: true,
		},
		{
			name:     "command with leading space",
			input:    " cat file.txt ",
			expected: true,
		},
		{
			name:     "ls with pipe and grep",
			input:    " ls -l | grep go",
			expected: true,
		},
		{
			name:     "cd with &&",
			input:    " cd /tmp && pwd",
			expected: true,
		},
		{
			name:     "date with format",
			input:    " date +%Y-%m-%d",
			expected: true,
		},
		{
			name:     "time with backticks containing invalid command",
			input:    "time `12:00`",
			expected: false,
		},
		{
			name:     "time with backticks containing valid command",
			input:    "time `date`",
			expected: true,
		},
		{
			name:     "who with wc",
			input:    " who | wc -l",
			expected: true,
		},
		{
			name:     "ps with grep",
			input:    " ps aux | grep go",
			expected: true,
		},
		{
			name:     "disk usage df",
			input:    " df -h",
			expected: true,
		},
		{
			name:     "disk usage du with wildcard",
			input:    " du -sh *",
			expected: true,
		},
		{
			name:     "cut with delimiter",
			input:    " cut -d, -f1",
			expected: true,
		},
		{
			name:     "awk with script",
			input:    " awk '{print $1}'",
			expected: true,
		},
		{
			name:     "sed with substitution",
			input:    " sed 's/foo/bar/g'",
			expected: true,
		},
		{
			name:     "tr with character set",
			input:    " tr 'a-z' 'A-Z'",
			expected: true,
		},
		{
			name:     "base64 decode",
			input:    " base64 -d",
			expected: true,
		},
		{
			name:     "curl with URL",
			input:    " curl -s URL",
			expected: true,
		},
		{
			name:     "wget with URL",
			input:    " wget -q URL",
			expected: true,
		},
		{
			name:     "vim with file",
			input:    " vim file.txt",
			expected: true,
		},
		{
			name:     "nano with file",
			input:    " nano file.txt",
			expected: true,
		},
		{
			name:     "less with file",
			input:    " less file.txt",
			expected: true,
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: false,
		},
		{
			name:     "command true",
			input:    "true",
			expected: true,
		},
		{
			name:     "command false",
			input:    "false",
			expected: true,
		},
		{
			name:     "command exit",
			input:    "exit",
			expected: true,
		},
		{
			name:     "command cd",
			input:    "cd",
			expected: true,
		},
		{
			name:     "command export",
			input:    "export",
			expected: true,
		},
		{
			name:     "command read",
			input:    "read",
			expected: true,
		},
		{
			name:     "command echo",
			input:    "echo",
			expected: true,
		},
		{
			name:     "git with URL",
			input:    "git remote add origin https://github.com/user/repo.git",
			expected: true,
		},
		{
			name:     "ssh with hostname and command",
			input:    "ssh user@server.com ls",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := CheckCommand(tt.input)
			if result != tt.expected {
				t.Errorf("CheckCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
