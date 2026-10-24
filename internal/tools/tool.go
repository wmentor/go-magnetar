package tools

import (
	"github.com/sashabaranov/go-openai"
)

// Tool is a single tool exposed to the LLM in the tool-use loop.
type Tool struct {
	// Name is the identifier of the tool (e.g., "web_search", "rag_search").
	Name string
	// Definition returns the OpenAI-compatible tool schema.
	Definition func() openai.Tool
	// Execute is called when the LLM invokes this tool.
	// args is the raw JSON arguments string.
	Execute ToolFn
	// IsSearchTool marks tools that count toward the per-request search call
	// limit (e.g. rag_search, web_search).
	IsSearchTool bool
	// Sequential marks tools that must NOT run concurrently with other tool
	// calls in the same LLM response. All sequential tools in a batch are
	// executed one after another (in the order the LLM listed them) after
	// all parallel tools have finished.
	Sequential bool
}

type ToolOption func(*Tool)

// GetName returns the name of the tool.
func (t Tool) GetName() string {
	return t.Name
}

func OptionSearch() ToolOption {
	return func(t *Tool) {
		if t != nil {
			t.IsSearchTool = true
		}
	}
}

func Sequential() ToolOption {
	return func(t *Tool) {
		if t != nil {
			t.Sequential = true
		}
	}
}
