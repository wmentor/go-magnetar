package editplugin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func init() {
	plugin.Register("chatcmd.edit", &Plugin{})
}

type Plugin struct {
	tools *confluence.ConfluenceTools
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	tools := confluence.New(s.Config, s)
	p.tools = tools

	if s.Config.Bool("confluence.enable") && s.Config.String("confluence.base_url") != "" {
		hub.RegisterChatCommand(plugin.ChatCommand{
			Name:    "confluence.edit",
			Help:    "Edit a Confluence page. Usage: /confluence.edit <url> <markdown_file>",
			Execute: p.execute,
		})
	}

	return nil
}

func (p *Plugin) execute(_ context.Context, _ plugin.AgentHandle, args string) error {
	if args == "" {
		fmt.Fprintln(os.Stdout, "Usage: /confluence.edit <url> <markdown_file>")
		return nil
	}

	// Split args into URL and markdown file
	parts := strings.Fields(args)
	if len(parts) < 2 {
		fmt.Fprintln(os.Stdout, "Usage: /confluence.edit <url> <markdown_file>")
		return nil
	}

	url := parts[0]
	markdownFile := parts[1]

	// Extract page ID from URL
	pageID, err := confluence.ExtractPageIDURL(url)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error extracting page ID from URL: %v\n", err)
		return nil
	}

	if strings.Contains(url, "/x/") || strings.Contains(url, "/p/") {
		pageID, err = confluence.ResolveShortPageID(pageID)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error resolve page ID from short URL: %v\n", err)
			return nil
		}
	}

	// Read markdown content from file
	markdownContent, err := os.ReadFile(markdownFile)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error reading markdown file %s: %v\n", markdownFile, err)
		return nil
	}

	if _, err = p.tools.EditPage(pageID, string(markdownContent)); err != nil {
		return err
	}

	return nil
}
