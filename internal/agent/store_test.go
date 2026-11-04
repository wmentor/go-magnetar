package agent

import (
	"slices"
	"testing"
)

func TestStore_New(t *testing.T) {
	t.Parallel()

	// Use existing testdata directories
	agents1Dir := "testdata/agents1"
	agents2Dir := "testdata/agents2"

	// Test with both directories
	store := NewStore(agents1Dir, agents2Dir)

	agents := store.List()

	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}

	// Check agent1
	agent1, err := store.Get("agent1")
	if err != nil {
		t.Fatalf("expected agent1 to exist: %v", err)
	}
	if len(agent1.Subagents) != 2 {
		t.Errorf("expected 2 subagents, got %d", len(agent1.Subagents))
	}

	// Check agent2
	agent2, err := store.Get("agent2")
	if err != nil {
		t.Fatalf("expected agent2 to exist: %v", err)
	}
	if !agent2.Disable {
		t.Error("expected agent2.Disable to be true")
	}

	// Check agent3
	agent3, err := store.Get("agent3")
	if err != nil {
		t.Fatalf("expected agent3 to exist: %v", err)
	}
	if agent3.Description != "Third agent from dir2" {
		t.Errorf("expected description 'Third agent from dir2', got '%s'", agent3.Description)
	}
}

func TestStore_Filter(t *testing.T) {
	t.Parallel()

	agents1Dir := "testdata/agents1"
	agents2Dir := "testdata/agents2"

	// Create store with multiple directories
	store := NewStore(agents1Dir, agents2Dir)

	tests := []struct {
		name         string
		filterAgents []string
		wantErr      bool
		wantCount    int
	}{
		{
			name:         "filter single agent",
			filterAgents: []string{"agent1"},
			wantErr:      false,
			wantCount:    1,
		},
		{
			name:         "filter multiple agents",
			filterAgents: []string{"agent1", "agent3"},
			wantErr:      false,
			wantCount:    2,
		},
		{
			name:         "filter all agents",
			filterAgents: []string{"agent1", "agent2", "agent3"},
			wantErr:      false,
			wantCount:    3,
		},
		{
			name:         "filter non-existent agent",
			filterAgents: []string{"nonexistent"},
			wantErr:      true,
			wantCount:    0,
		},
		{
			name:         "filter mix of existent and non-existent",
			filterAgents: []string{"agent1", "nonexistent"},
			wantErr:      true,
			wantCount:    1,
		},
		{
			name:         "filter empty list",
			filterAgents: []string{},
			wantErr:      false,
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			filtered, err := store.Filter(tt.filterAgents)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if filtered == nil {
				t.Fatal("expected non-nil filtered store")
			}

			if len(filtered.List()) != tt.wantCount {
				t.Errorf("expected %d agents, got %d", tt.wantCount, len(filtered.List()))
			}

			// Verify agents are deep copied (modifying filtered doesn't affect original)
			if tt.wantCount > 0 {
				agentNames := make([]string, 0, tt.wantCount)
				for _, agent := range filtered.List() {
					agentNames = append(agentNames, agent.Name)
				}

				for _, name := range tt.filterAgents {
					found := slices.Contains(agentNames, name)
					if !found {
						t.Errorf("filtered store missing agent: %s", name)
					}
				}
			}
		})
	}
}

func TestStore_New_WithSubagents(t *testing.T) {
	t.Parallel()

	// Use existing testdata directory (agents2Dir has agent3.md with 1 subagent)
	agentsDir := "testdata/agents2"

	store := NewStore(agentsDir)
	agents := store.List()

	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}

	// Check agent3 has 1 subagent
	agent3, err := store.Get("agent3")
	if err != nil {
		t.Fatalf("expected agent3 to exist: %v", err)
	}
	if len(agent3.Subagents) != 1 {
		t.Errorf("expected 1 subagent, got %d", len(agent3.Subagents))
	}
}
