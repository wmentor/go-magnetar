package agentplugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/agent/subagent"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.agent", &Plugin{})
}

type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	p.state = state
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "agent",
		Aliases: []string{},
		Help:    "List available subagents or switch by name. Usage: /agent [agent_name] [request]",
		Execute: p.execute,
	})
	return nil
}

func (p *Plugin) execute(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	args := strings.TrimSpace(params.Args)
	if args == "" {
		return &plugin.ChatCommandResponse{
			Stdout: listAgents(p.state.AgentStore),
			NoLog:  true,
		}, nil
	}

	// Split agent name from request
	parts := splitArgs(args, 2)
	if len(parts) == 0 || parts[0] == "" {
		return &plugin.ChatCommandResponse{
			Stdout: "Error: agent name is required\n",
			NoLog:  true,
		}, nil
	}

	agentName := parts[0]
	request := ""
	if len(parts) > 1 {
		request = parts[1]
	}

	// Create subagent
	sa, err := subagent.New(agentName, p.state)
	if err != nil {
		return &plugin.ChatCommandResponse{
			Stdout: fmt.Sprintf("Error creating subagent %q: %v\n", agentName, err),
			NoLog:  true,
		}, nil
	}

	// Execute request if provided
	var response string
	if request != "" {
		response, err = sa.Ask(request)
		if err != nil {
			return &plugin.ChatCommandResponse{
				Stdout: fmt.Sprintf("Error asking subagent %q: %v\n", agentName, err),
				NoLog:  true,
			}, nil
		}
	} else {
		response = fmt.Sprintf("Subagent %q initialized. You can now interact with it.\n", agentName)
	}

	return &plugin.ChatCommandResponse{
		Stdout: response,
		NoLog:  true,
	}, nil
}

func listAgents(agentStore plugin.AgentStore) string {
	var sb strings.Builder
	sb.WriteString("Available agents:\n")
	if agentStore == nil {
		return sb.String()
	}

	for _, agent := range agentStore.List() {
		if agent.Disable {
			continue
		}
		if agent.Description != "" {
			fmt.Fprintf(&sb, "  * %s - %s\n", agent.Name, agent.Description)
		} else {
			fmt.Fprintf(&sb, "  * %s\n", agent.Name)
		}
	}
	return sb.String()
}

// splitArgs splits the input string into at most n parts, separated by a space.
// It returns a slice of at least length 1.
func splitArgs(s string, n int) []string {
	result := make([]string, 0, n)
	start := 0
	count := 0

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if count == 0 {
				result = append(result, s[start:i])
				count++
				start = i + 1
			} else if count < n-1 {
				result = append(result, s[start:i])
				count++
				start = i + 1
			}
		}
	}
	result = append(result, s[start:])

	// Ensure at least one element
	if len(result) == 0 {
		result = []string{""}
	}

	return result
}
