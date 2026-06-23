package main

import (
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/cmd"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/compact"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/exit"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/help"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/new"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/save"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/stat"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/version"
	_ "github.com/wmentor/go-magnetar/internal/plugins/cli/agent"
	_ "github.com/wmentor/go-magnetar/internal/plugins/cli/indexer"
	_ "github.com/wmentor/go-magnetar/internal/plugins/generic"
	_ "github.com/wmentor/go-magnetar/internal/plugins/rag"
	_ "github.com/wmentor/go-magnetar/internal/plugins/web"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
