package ask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

func init() {
	plugin.Register("ask", &Plugin{})
}

type Plugin struct{}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	hub.RegisterTool(p.createTool())
	return nil
}

func (p *Plugin) createTool() plugin.LLMTool {
	return plugin.LLMTool{
		Definition: func() openai.Tool {
			return openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "ask",
					Description: "Ask the user a clarifying question to obtain additional information",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"question": map[string]any{
								"type":        "string",
								"description": "Question to ask the user",
							},
						},
						"required": []string{"question"},
					},
				},
			}
		},
		Execute: func(ctx context.Context, args string) (string, error) {
			type askArgs struct {
				Question string `json:"question"`
			}

			var a askArgs
			if err := json.Unmarshal([]byte(args), &a); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			return printer.Ask(a.Question), nil
		},
		IsSearchTool: false,
	}
}
