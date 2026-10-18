package chat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func TestLoadProjectContext(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		expectedName string
		expectedBody string
		expectedOk   bool
	}{
		{
			name:         "only AGENTS.md",
			files:        map[string]string{"AGENTS.md": "agents"},
			expectedName: "AGENTS.md",
			expectedBody: "agents",
			expectedOk:   true,
		},
		{
			name:         "only CLAUDE.md",
			files:        map[string]string{"CLAUDE.md": "claude"},
			expectedName: "CLAUDE.md",
			expectedBody: "claude",
			expectedOk:   true,
		},
		{
			name:         "both files, AGENTS.md wins",
			files:        map[string]string{"AGENTS.md": "agents", "CLAUDE.md": "claude"},
			expectedName: "AGENTS.md",
			expectedBody: "agents",
			expectedOk:   true,
		},
		{
			name:       "neither file",
			files:      map[string]string{},
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
			}

			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatalf("open root: %v", err)
			}
			defer root.Close()

			name, content, ok := loadProjectContext(root)
			if ok != tt.expectedOk || name != tt.expectedName || string(content) != tt.expectedBody {
				t.Errorf("loadProjectContext() = (%q, %q, %v); want (%q, %q, %v)",
					name, content, ok, tt.expectedName, tt.expectedBody, tt.expectedOk)
			}
		})
	}

	t.Run("nil root", func(t *testing.T) {
		if _, _, ok := loadProjectContext(nil); ok {
			t.Errorf("loadProjectContext(nil) ok = true; want false")
		}
	})
}

func TestMatchCommand(t *testing.T) {
	tests := []struct {
		name      string
		nameInput string
		cmd       plugin.ChatCommand
		expected  bool
	}{
		{
			name:      "match exact name (case-insensitive)",
			nameInput: "help",
			cmd:       plugin.ChatCommand{Name: "Help", Aliases: []string{}},
			expected:  true,
		},
		{
			name:      "match exact name lowercase",
			nameInput: "help",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{}},
			expected:  true,
		},
		{
			name:      "match exact name uppercase",
			nameInput: "HELP",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{}},
			expected:  true,
		},
		{
			name:      "no match with different name",
			nameInput: "exit",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{}},
			expected:  false,
		},
		{
			name:      "match first alias",
			nameInput: "h",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{"h", "halp"}},
			expected:  true,
		},
		{
			name:      "match second alias",
			nameInput: "halp",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{"h", "halp"}},
			expected:  true,
		},
		{
			name:      "match alias case-insensitive",
			nameInput: "HALP",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{"h", "halp"}},
			expected:  true,
		},
		{
			name:      "no match with alias",
			nameInput: "helpme",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{"h"}},
			expected:  false,
		},
		{
			name:      "no aliases",
			nameInput: "help",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{}},
			expected:  true,
		},
		{
			name:      "empty name string, match empty command",
			nameInput: "",
			cmd:       plugin.ChatCommand{Name: "", Aliases: []string{}},
			expected:  true,
		},
		{
			name:      "empty name string, no match",
			nameInput: "",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{}},
			expected:  false,
		},
		{
			name:      "empty name with empty alias",
			nameInput: "",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{""}},
			expected:  true,
		},
		{
			name:      "match with multiple aliases",
			nameInput: "h",
			cmd:       plugin.ChatCommand{Name: "help", Aliases: []string{"a", "b", "h", "c"}},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchCommand(tt.nameInput, tt.cmd)
			if result != tt.expected {
				t.Errorf("matchCommand(%q, {%q, %v}) = %v; want %v",
					tt.nameInput, tt.cmd.Name, tt.cmd.Aliases, result, tt.expected)
			}
		})
	}
}
