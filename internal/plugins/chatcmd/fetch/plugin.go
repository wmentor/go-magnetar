package fetchplugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wmentor/go-magnetar/internal/common"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/web/webfetch"
)

func init() {
	plugin.Register("chatcmd.fetch", &Plugin{})
}

const (
	timeout = time.Second * 30
)

type Plugin struct {
	tool *webfetch.Tool
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.tool = webfetch.NewTool(s.Config, s.Printer, s.RequesterFactory, s)

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "fetch",
		Help:    "Fetch content from a URL. Usage: /fetch <url> [file]",
		Aliases: []string{"f"},
		Execute: p.execute,
	})

	return nil
}

func (p *Plugin) execute(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := params.Args
	if args == "" {
		fmt.Fprintln(os.Stdout, "Usage: /fetch <url> [file]")
		return plugin.NopChatCommandResponse, nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		fmt.Fprintln(os.Stdout, "Usage: /fetch <url> [file]")
		return plugin.NopChatCommandResponse, nil
	}

	url := parts[0]
	var filename string
	if len(parts) >= 2 {
		filename = outputPath(parts[1])
	}

	content, err := p.tool.WebFetchEngine(ctx, url, true)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error fetching URL: %v\n", err)
		return plugin.NopChatCommandResponse, nil
	}

	if filename != "" {
		if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stdout, "Error saving file: %v\n", err)
			return plugin.NopChatCommandResponse, nil
		}
		fmt.Fprintf(os.Stdout, "Content saved to %s\n", filename)
		return plugin.NopChatCommandResponse, nil
	}

	if p.checkLess() {
		stdin := bytes.NewBuffer(nil)
		stdin.WriteString(content)

		cmd := exec.Command("less")
		cmd.Stdin = stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stdout, "Error running less: %v\n", err)
			fmt.Fprintln(os.Stdout, content)
		}
	} else {
		fmt.Fprintln(os.Stdout, content)
	}

	return plugin.NopChatCommandResponse, nil
}

// outputPath cleans the file argument of /fetch and expands a leading ~ to the home directory.
func outputPath(arg string) string {
	return common.ExpandHome(filepath.Clean(arg))
}

func (p *Plugin) checkLess() bool {
	_, err := exec.LookPath("less")
	return err == nil
}
