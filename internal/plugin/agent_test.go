package plugin

import (
	"slices"
	"testing"
)

func TestAgent_Copy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		agent *Agent
	}{
		{
			name:  "nil agent",
			agent: nil,
		},
		{
			name: "agent with empty fields",
			agent: &Agent{
				Name:           "",
				Description:    "",
				Prompt:         "",
				AvailableTools: nil,
				Disable:        false,
				Subagents:      nil,
			},
		},
		{
			name: "agent with all fields set",
			agent: &Agent{
				Name:           "test-agent",
				Description:    "Test agent description",
				Prompt:         "You are a helpful assistant.",
				AvailableTools: []string{"web_search", "file_read", "cve"},
				Disable:        false,
				Subagents:      []string{"subagent1", "subagent2"},
			},
		},
		{
			name: "agent with empty slices",
			agent: &Agent{
				Name:           "test-agent",
				Description:    "Test agent description",
				Prompt:         "You are a helpful assistant.",
				AvailableTools: []string{},
				Disable:        false,
				Subagents:      []string{},
			},
		},
		{
			name: "agent with Disable flag true",
			agent: &Agent{
				Name:           "disabled-agent",
				Description:    "Disabled agent",
				Prompt:         "You are disabled.",
				AvailableTools: []string{},
				Disable:        true,
				Subagents:      []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			result := tt.agent.Copy()

			if tt.agent == nil {
				if result != nil {
					t.Errorf("Copy() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Copy() returned nil, expected non-nil")
			}

			// Check that result is a different pointer
			if result == tt.agent {
				t.Errorf("Copy() returned same pointer, expected deep copy")
			}

			// Check all fields
			if result.Name != tt.agent.Name {
				t.Errorf("Copy().Name = %v, want %v", result.Name, tt.agent.Name)
			}
			if result.Description != tt.agent.Description {
				t.Errorf("Copy().Description = %v, want %v", result.Description, tt.agent.Description)
			}
			if result.Prompt != tt.agent.Prompt {
				t.Errorf("Copy().Prompt = %v, want %v", result.Prompt, tt.agent.Prompt)
			}
			if result.Disable != tt.agent.Disable {
				t.Errorf("Copy().Disable = %v, want %v", result.Disable, tt.agent.Disable)
			}

			// Check slices are deeply copied
			if !slices.Equal(result.AvailableTools, tt.agent.AvailableTools) {
				t.Errorf("Copy().AvailableTools = %v, want %v", result.AvailableTools, tt.agent.AvailableTools)
			}

			// Verify it's a deep copy by modifying the copy and checking original is unchanged
			result.AvailableTools = append(result.AvailableTools, "new_tool")
			result.Subagents = append(result.Subagents, "new_subagent")

			if len(result.AvailableTools) == len(tt.agent.AvailableTools) {
				t.Errorf("Original AvailableTools was modified, expected deep copy")
			}
			if len(result.Subagents) == len(tt.agent.Subagents) {
				t.Errorf("Original Subagents was modified, expected deep copy")
			}
		})
	}
}
