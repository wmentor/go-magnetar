package indexcmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "idxtab",
		Aliases: []string{},
		Help:    "Index multiple files/URLs from a JSON lines file (one per line, format: {\"source\":\"path|url\",\"message\":\"text\"})",
		Execute: p.executeTab,
	})

	return nil
}

func (p *Plugin) execute(ctx context.Context, agent plugin.AgentHandle, args string) error {
	if args == "" {
		return fmt.Errorf("usage: /index <path|url> [-m <message>]")
	}

	target := strings.TrimSpace(args)
	message := ""

	parts := strings.Fields(args)
	target = parts[0]
	for i := 1; i < len(parts); i++ {
		if parts[i] == "-m" && i+1 < len(parts) {
			message = parts[i+1]
			i++
		}
	}

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		slog.Info("indexcmd: indexing URL", "url", target)
		if err := p.idx.IndexURL(target, message); err != nil {
			return fmt.Errorf("index URL failed: %w", err)
		}
		return nil
	}

	slog.Info("indexcmd: indexing file", "file", target)
	if err := p.idx.IndexFile(target, message); err != nil {
		return fmt.Errorf("index file failed: %w", err)
	}
	return nil
}

type Entry struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

func (p *Plugin) executeTab(ctx context.Context, agent plugin.AgentHandle, args string) error {
	filename := strings.TrimSpace(args)

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file error: %w", err)
	}
	defer file.Close()

	workDir := filepath.Dir(filename)

	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			slog.Error("json parse error", "line", line, "error", err)
			continue
		}

		source := entry.Source
		message := entry.Message

		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			slog.Info("indexcmd: indexing URL from tab", "url", source)
			if err := p.idx.IndexURL(source, message); err != nil {
				slog.Error("index URL failed", "url", source, "error", err)
				continue
			}
		} else {
			if !filepath.IsAbs(source) {
				source = filepath.Join(workDir, source)
			}
			slog.Info("indexcmd: indexing file from tab", "file", source)
			if err := p.idx.IndexFile(source, message); err != nil {
				slog.Error("index file failed", "file", source, "error", err)
				continue
			}
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	return nil
}
