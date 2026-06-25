package indexcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/wmentor/go-magnetar/internal/agent/indexer"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

func init() {
	plugin.Register("indexcmd", &Plugin{})
}

type Plugin struct {
	idx *indexer.Indexer
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	root, err := os.OpenRoot("/")
	if err != nil {
		return fmt.Errorf("indexcmd: failed to open root: %w", err)
	}
	defer root.Close()

	ragTools, err := rag.New(s.Config)
	if err != nil {
		return fmt.Errorf("indexcmd: failed to init RAG: %w", err)
	}

	webTools, err := web.New(s.Config, root)
	if err != nil {
		return fmt.Errorf("indexcmd: failed to init web: %w", err)
	}

	p.idx, err = indexer.New(s.Config, root)
	if err != nil {
		return fmt.Errorf("indexcmd: failed to create indexer: %w", err)
	}
	_ = ragTools
	_ = webTools

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "index",
		Aliases: []string{"i"},
		Help:    "Index file or URL into RAG knowledge base (auto-detects URL vs file)",
		Execute: p.execute,
	})

	return nil
}

func (p *Plugin) execute(ctx context.Context, agent plugin.AgentHandle, args string) error {
	if args == "" {
		return fmt.Errorf("usage: /index <path|url> [-m <message>]")
	}

	target := strings.TrimSpace(args)
	message := ""

	// Parse optional -m flag
	parts := strings.Fields(args)
	target = parts[0]
	for i := 1; i < len(parts); i++ {
		if parts[i] == "-m" && i+1 < len(parts) {
			message = parts[i+1]
			i++
		}
	}

	// Auto-detect URL vs file
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		slog.Info("indexcmd: indexing URL", "url", target)
		if err := p.idx.IndexURL(target, message); err != nil {
			return fmt.Errorf("index URL failed: %w", err)
		}
		return nil
	}

	// Default to file for anything else (relative/absolute paths)
	slog.Info("indexcmd: indexing file", "file", target)
	if err := p.idx.IndexFile(target, message); err != nil {
		return fmt.Errorf("index file failed: %w", err)
	}
	return nil
}
