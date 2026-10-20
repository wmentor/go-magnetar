package lastresponse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

const lastResponseToolName = "last_response"

func init() {
	plugin.Register("last_response", &Plugin{})
}

type Plugin struct{}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	// Register the last_response tool
	hub.RegisterTool(plugin.LLMTool{
		Definition: func() openai.Tool {
			return openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name: lastResponseToolName,
					Description: `Retrieve a previously stored response from the conversation history.
This tool allows you to access previous answers to the same question or related questions.
Use this tool ONLY when explicitly requested by the user.
For example:
- "What was your last answer?"
- "Show me the previous response about X"
- "What did you say before?"
- "Retrieve the last response"`,
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"index": map[string]any{
								"type":        "integer",
								"description": "The index of the response to retrieve (0 = most recent, 1 = second most recent, etc.)",
							},
						},
						"required": []string{"index"},
					},
				},
			}
		},
		Execute: func(ctx context.Context, args string) (string, error) {
			var params struct {
				Index int `json:"index"`
			}

			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}

			// Validate index
			if params.Index < 0 {
				return "", fmt.Errorf("index must be non-negative")
			}

			// Get response from store
			response, err := s.ResponseStore.Last(params.Index)
			if err != nil {
				return "", fmt.Errorf("no response found at the specified index. Try a smaller index value")
			}

			// Format the response with metadata
			return fmt.Sprintf("Response at index %d (most recent = 0):\n\n%s", params.Index, response), nil
		},
	})

	return nil
}
