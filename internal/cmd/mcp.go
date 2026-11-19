package cmd

import (
	"context"
	"fmt"
	"net/http"
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
		Name:    "magnetar",
		Title:   "magnetar",
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

	mode := state.Config.String("mcp.mode")
	if mode != "stdio" && mode != "http" {
		fmt.Fprintln(os.Stderr, "invalid value of mcp.mode")
		os.Exit(1)
	}

	if mode == "stdio" {
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(os.Stderr, "MCP stdio server failed to start: %v\n", err)
			os.Exit(1)
		}
		return
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)

	listen := fmt.Sprintf("%s:%d", state.Config.String("mcp.server.host"), state.Config.Int("mcp.server.port"))

	if state.Config.Bool("mcp.server.tls") {
		if err := http.ListenAndServeTLS(listen, state.Config.String("mcp.server.cert"),
			state.Config.String("mcp.server.key"), mux); err != nil {
			fmt.Fprintf(os.Stderr, "MCP HTTPS server failed to start on %s: %v\n", listen, err)
			os.Exit(1)
		}
	} else {
		if err := http.ListenAndServe(listen, mux); err != nil {
			fmt.Fprintf(os.Stderr, "MCP HTTP server failed to start on %s: %v\n", listen, err)
			os.Exit(1)
		}
	}
}
