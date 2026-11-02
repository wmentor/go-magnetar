package agent

import (
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
