package chat

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

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
