package cmd

import (
	"context"
	"os"
	"runtime/debug"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func mcpFlow(state *plugin.State) {
	version := "devel"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "go-magnetar",
		Title:   "go-magnetar",
		Version: version,
	}, nil)

	for _, tool := range state.ToolStore.OutToolsMCP() {
		server.AddTool(
			&mcp.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			},
			tool.Exec,
		)
	}

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		os.Exit(1)
	}
}
