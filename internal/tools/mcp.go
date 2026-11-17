package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ToolMCP struct {
	// Name is the identifier of the tool (e.g., "web_search", "rag_search").
	Name string
	// Description is the description of the tool.
	Description string
	// InputSchema input params schema.
	InputSchema json.RawMessage
	// Execute is called when the LLM invokes this tool.
	Exec func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// toolErr converts an engine error into a tool-level error result (IsError=true).
// Per MCP spec, tool errors must be returned inside CallToolResult so the LLM
// can observe them and self-correct; returning (nil, err) would produce a
// protocol-level JSON-RPC error that the LLM never sees.
func mcpOurToolErr(err error) (*mcp.CallToolResult, error) {
	res := &mcp.CallToolResult{}
	res.SetError(err)
	return res, nil
}

// resultText wraps a plain string in a successful tool result.
func mcpOurResultText(s string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: s}},
	}, nil
}

// parseArgs deserialises the tool request arguments into dst.
// Returns a tool-level error if the payload is malformed.
func mcpOurParseArgs(req *mcp.CallToolRequest, dst any) (*mcp.CallToolResult, error) {
	if req.Params.Arguments == nil {
		return nil, nil //nolint:nilnil // no arguments — caller uses zero values.
	}
	if err := json.Unmarshal(req.Params.Arguments, dst); err != nil {
		return mcpOurToolErr(fmt.Errorf("invalid arguments: %w", err))
	}
	return nil, nil //nolint:nilnil // ok.
}
