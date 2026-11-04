package agentplugin

import (
	"context"
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

type listingAgentStore struct {
	agents []*plugin.Agent
}

func (s *listingAgentStore) Get(string) (*plugin.Agent, error) {
	return nil, plugin.ErrAgentNotFound
}

func (s *listingAgentStore) List() []*plugin.Agent {
	return s.agents
}

func (s *listingAgentStore) Filter([]string) (plugin.AgentStore, error) {
	return s, nil
}

func TestAgentPlugin_ListAvailableAgents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     string
		agents   []*plugin.Agent
		nilStore bool
		want     string
	}{
		{
			name: "agent with description",
			agents: []*plugin.Agent{
				{Name: "research", Description: "Research assistant"},
			},
			want: "Available agents:\n  * research - Research assistant\n",
		},
		{
			name: "whitespace arguments",
			args: " \t\n ",
			agents: []*plugin.Agent{
				{Name: "research", Description: "Research assistant"},
			},
			want: "Available agents:\n  * research - Research assistant\n",
		},
		{
			name: "agent without description",
			agents: []*plugin.Agent{
				{Name: "coding"},
			},
			want: "Available agents:\n  * coding\n",
		},
		{
			name: "disabled agents omitted and store order preserved",
			agents: []*plugin.Agent{
				{Name: "coding", Description: "Programming assistant"},
				{Name: "hidden", Description: "Disabled assistant", Disable: true},
				{Name: "writing"},
			},
			want: "Available agents:\n  * coding - Programming assistant\n  * writing\n",
		},
		{
			name: "only disabled agents",
			agents: []*plugin.Agent{
				{Name: "hidden", Disable: true},
			},
			want: "Available agents:\n",
		},
		{
			name:   "empty store",
			agents: []*plugin.Agent{},
			want:   "Available agents:\n",
		},
		{
			name: "nil agent list",
			want: "Available agents:\n",
		},
		{
			name:     "store not configured",
			nilStore: true,
			want:     "Available agents:\n",
		},
		{
			name: "named agent still uses the lookup path",
			args: "missing Explain this",
			want: "Error creating subagent \"missing\": get agent \"missing\" error: agent not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()

			state := &plugin.State{}
			if !tt.nilStore {
				state.AgentStore = &listingAgentStore{agents: tt.agents}
			}
			hub := &mockHub{}
			p := &Plugin{}
			if err := p.Init(state, hub); err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			response, err := hub.commands[0].Execute(context.Background(), &plugin.ChatCommandInput{Args: tt.args})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if response.Stdout != tt.want {
				t.Errorf("Stdout = %q, want %q", response.Stdout, tt.want)
			}
			if !response.NoLog || response.Content != "" || response.Exit {
				t.Errorf("listing must only display stdout without logging, exiting, or sending model content: %+v", response)
			}
		})
	}
}
