package fetchplugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

func init() {
	plugin.Register("chatcmd.fetch", &Plugin{})
}

type Plugin struct {
	tools *web.WebTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	webTools, err := web.New(s.Config, s.Root)
	if err != nil {
		return fmt.Errorf("indexer: failed to initialise web tools: %w", err)
	}

	p.tools = webTools

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "fetch",
		Help:    "Fetch content from a URL. Usage: /fetch <url> [file]",
		Aliases: []string{"f"},
		Execute: p.execute,
	})

	return nil
}

func (p *Plugin) execute(_ context.Context, a plugin.AgentHandle, args string) error {
	if args == "" {
		fmt.Fprintln(os.Stdout, "Usage: /fetch <url> [file]")
		return nil
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		fmt.Fprintln(os.Stdout, "Usage: /fetch <url> [file]")
		return nil
	}

	url := parts[0]
	var filename string
	if len(parts) >= 2 {
		filename = filepath.Clean(parts[1])
	}

	var content string
	var err error

	if p.tools != nil {
		content, err = p.tools.WebFetch(url)
	} else {
		fmt.Fprintln(os.Stdout, "Error: webfetch is not configured")
		return nil
	}

	if err != nil {
		fmt.Fprintf(os.Stdout, "Error fetching URL: %v\n", err)
		return nil
	}

	if filename != "" {
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stdout, "Error saving file: %v\n", err)
			return nil
		}
		fmt.Fprintf(os.Stdout, "Content saved to %s\n", filename)
		return nil
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

	return nil
}

func (p *Plugin) checkLess() bool {
	_, err := exec.LookPath("less")
	return err == nil
}
