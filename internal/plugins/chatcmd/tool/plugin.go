package toolplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/wmentor/go-magnetar/internal/gmutex"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const usageMessage = "Usage: /tool <tool_name> [json_params]"

func init() {
	plugin.Register("chatcmd.tool", &Plugin{})
}

// Plugin registers the /tool chat command.
type Plugin struct {
	state *plugin.State
}

// Init initializes the plugin and registers the /tool chat command.
func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	p.state = state

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "tool",
		Aliases: nil,
		Help:    "Execute a registered tool with JSON parameters. " + usageMessage,
		Execute: p.execute,
	})

	return nil
}

func (p *Plugin) execute(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	if p.state == nil || p.state.ToolStore == nil {
		return errorResponse("Error: tool store is unavailable."), nil
	}

	args := ""
	if params != nil {
		args = params.Args
	}

	toolName, rawJSON, ok := parseToolArgs(args)
	if !ok {
		return errorResponse(usageMessage), nil
	}

	if err := validateJSONObject(rawJSON); err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	tool, err := p.state.ToolStore.GetTool(toolName)
	if err != nil {
		if errors.Is(err, tools.ErrToolNotFound) {
			return errorResponse(fmt.Sprintf("Error: tool %q not found. Use /tools to list available tools.", toolName)), nil
		}
		return errorResponse(fmt.Sprintf("Error looking up tool %q: %v", toolName, err)), nil
	}

	if tool.Execute == nil {
		return errorResponse(fmt.Sprintf("Error: tool %q has no executable handler.", toolName)), nil
	}

	rawOutput := executeTool(ctx, tool, []byte(rawJSON))
	stdout, noLog := formatToolOutput(toolName, rawOutput)

	return &plugin.ChatCommandResponse{
		Stdout: stdout,
		NoLog:  noLog,
	}, nil
}

func parseToolArgs(args string) (string, string, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return "", "", false
	}

	idx := strings.IndexFunc(trimmed, unicode.IsSpace)
	if idx == -1 {
		return trimmed, "{}", true
	}

	toolName := trimmed[:idx]
	jsonParams := strings.TrimSpace(trimmed[idx:])
	if jsonParams == "" {
		jsonParams = "{}"
	}

	return toolName, jsonParams, true
}

func validateJSONObject(rawJSON string) error {
	var obj map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &obj); err != nil {
		return fmt.Errorf("invalid JSON parameters: %w", err)
	}
	if obj == nil {
		return errors.New("JSON parameters must be a JSON object")
	}
	return nil
}

func executeTool(ctx context.Context, tool tools.Tool, rawJSON []byte) string {
	if !tool.IsAgent {
		if tool.Sequential {
			gmutex.Lock()
			defer gmutex.Unlock()
		} else {
			gmutex.RLock()
			defer gmutex.RUnlock()
		}
	}

	return tool.Execute(ctx, rawJSON)
}

func formatToolOutput(toolName, rawOutput string) (string, bool) {
	trimmed := strings.TrimSpace(rawOutput)
	if trimmed == "" {
		return fmt.Sprintf("Tool %q completed with empty output.\n", toolName), false
	}

	var resp tools.Response
	if err := json.Unmarshal([]byte(trimmed), &resp); err == nil {
		if len(resp.Errors) > 0 {
			return formatToolErrors(toolName, resp.Errors), true
		}
		if resp.Status == tools.StatusOK {
			if resp.Result == "" {
				return fmt.Sprintf("Tool %q completed with empty output.\n", toolName), false
			}
			return resp.Result, false
		}
	}

	if strings.HasPrefix(trimmed, "\"") {
		var str string
		if err := json.Unmarshal([]byte(trimmed), &str); err == nil {
			if str == "" {
				return fmt.Sprintf("Tool %q completed with empty output.\n", toolName), false
			}
			return str, false
		}
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(trimmed), "", "  "); err == nil {
		var sb strings.Builder
		sb.WriteString("```json\n")
		sb.Write(indented.Bytes())
		sb.WriteString("\n```\n")
		return sb.String(), resp.Status == "validation_failed"
	}

	return rawOutput, false
}

func formatToolErrors(toolName string, errs map[string]string) string {
	keys := make([]string, 0, len(errs))
	for k := range errs {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var sb strings.Builder
	fmt.Fprintf(&sb, "Error executing tool %q:\n", toolName)
	for _, k := range keys {
		fmt.Fprintf(&sb, "- %s: %s\n", k, errs[k])
	}

	return sb.String()
}

func errorResponse(msg string) *plugin.ChatCommandResponse {
	return &plugin.ChatCommandResponse{
		Stdout: msg + "\n",
		NoLog:  true,
	}
}
