package generic

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/wmentor/go-magnetar/internal/config"
)

// GenericTools provides basic file system operations as LLM tools.
type GenericTools struct {
	cfg *config.Config
}

// New creates a new GenericTools instance.
func New(cfg *config.Config) *GenericTools {
	return &GenericTools{cfg: cfg}
}

// FileRead reads a file and returns its content and a success flag.
func (g *GenericTools) FileRead(filename string) (string, bool) {
	data, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("file_read: failed to read file", "file", filename, "err", err)
		return "", false
	}
	return string(data), true
}

// Definition returns the OpenAI tool schema for file_read.
func (g *GenericTools) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_read",
			Description: "Read the contents of a file by its path",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"filename"},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (g *GenericTools) Dispatch(name string, args string) string {
	switch name {
	case "file_read":
		var params struct {
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("file_read: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		content, ok := g.FileRead(params.Filename)
		if !ok {
			return "error: failed to read file"
		}
		return content
	default:
		return "error: unknown tool " + name
	}
}
