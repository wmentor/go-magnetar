package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/wmentor/go-magnetar/internal/cmd"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/compact"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/exit"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/fetch"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/help"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/less"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/new"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/readonly"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/stat"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/version"
	_ "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/write"
	_ "github.com/wmentor/go-magnetar/internal/plugins/generic"
	_ "github.com/wmentor/go-magnetar/internal/plugins/github"
	_ "github.com/wmentor/go-magnetar/internal/plugins/gitlab"
	_ "github.com/wmentor/go-magnetar/internal/plugins/indexcmd"
	_ "github.com/wmentor/go-magnetar/internal/plugins/jira"
	_ "github.com/wmentor/go-magnetar/internal/plugins/rag"
	_ "github.com/wmentor/go-magnetar/internal/plugins/web"
)

const (
	userRoot   = "root"
	userRootID = "0"
)

func main() {
	usr, err := user.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "get current user info failed")
		os.Exit(1)
	}

	if usr.Username == userRoot || usr.Uid == userRootID {
		fmt.Fprintln(os.Stderr, "go-magnetar can't be run as superuser")
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
