package plugin

import (
	"errors"
	"slices"
)

var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrAgentName     = errors.New("agent name not found in store")
)

// Agent represents a chat agent with its configuration and state.
type Agent struct {
	Name           string
	Description    string
	Prompt         string
	AvailableTools []string
	Disable        bool
	Subagents      []string
}

// Copy creates a deep copy of the Agent.
func (a *Agent) Copy() *Agent {
	if a == nil {
		return nil
	}

	return &Agent{
		Name:           a.Name,
		Description:    a.Description,
		Prompt:         a.Prompt,
		AvailableTools: slices.Clone(a.AvailableTools),
		Disable:        a.Disable,
		Subagents:      slices.Clone(a.Subagents),
	}
}

type AgentStore interface {
	Get(agentName string) (*Agent, error)
	List() []*Agent
	Filter(agents []string) (AgentStore, error)
}
