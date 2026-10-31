package indexcmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wmentor/go-magnetar/internal/agent/indexer"
	"github.com/wmentor/go-magnetar/internal/common"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

func init() {
	plugin.Register("indexcmd", &Plugin{})
}

type Plugin struct {
	idx *indexer.Indexer
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	if !s.Config.Bool("rag.enable") || s.Config.String("rag.llm.base_url") == "" {
		return nil
	}

	root, err := os.OpenRoot("/")
	if err != nil {
		return fmt.Errorf("indexcmd: failed to open root: %w", err)
	}
	defer root.Close()

	p.idx, err = indexer.New(s.Config, root, s)
	if err != nil {
		return fmt.Errorf("indexcmd: failed to create indexer: %w", err)
	}

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "index",
		Aliases: []string{"i"},
		Help:    "Index a file, URL, or directory into RAG knowledge base (use `-r` for directories)",
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

func (p *Plugin) execute(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	args := params.Args
	if args == "" {
		return nil, fmt.Errorf("usage: /index <path|url> [-r|--recursive] [-m <message>]")
	}

	opts, err := parseIndexArgs(args)
	if err != nil {
		return nil, fmt.Errorf("parse index arg error: %w", err)
	}

	switch {
	case isURL(opts.target):
		if err := p.idx.IndexURL(opts.target, opts.message); err != nil {
			return nil, fmt.Errorf("index URL failed: %w", err)
		}
		return plugin.NopChatCommandResponse, nil
	case isDir(opts.target):
		if !opts.recursive {
			return nil, fmt.Errorf("target is a directory, use -r or --recursive for recursive indexing")
		}

		err := p.idx.IndexDirectory(opts.target, opts.message)
		if err != nil {
			return nil, fmt.Errorf("index directory failed: %w", err)
		}
		return plugin.NopChatCommandResponse, nil
	default:
		if err := p.idx.IndexFile(opts.target, opts.message); err != nil {
			return nil, fmt.Errorf("index file failed: %w", err)
		}
	}

	return plugin.NopChatCommandResponse, nil
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func parseIndexArgs(args string) (IndexArgs, error) {
	parts := strings.Fields(args)
	opts := IndexArgs{}

	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "-m":
			if i+1 < len(parts) {
				opts.message = parts[i+1]
				i++
			} else {
				return IndexArgs{}, fmt.Errorf("message can not be empty")
			}
		case "-r", "--recursive":
			opts.recursive = true
		default:
			if strings.HasPrefix(parts[i], "-") {
				return IndexArgs{}, fmt.Errorf("unknown flag: %q", parts[i])
			}
			if opts.target != "" {
				return IndexArgs{}, fmt.Errorf("too many targets: %q and %q", opts.target, parts[i])
			}
			opts.target = parts[i]
		}
	}

	if opts.target == "" {
		return IndexArgs{}, fmt.Errorf("target not found")
	}
	return opts, nil
}

type IndexArgs struct {
	target    string
	message   string
	recursive bool
}

type Entry struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

func (p *Plugin) executeTab(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	args := params.Args
	filename := common.ExpandHome(strings.TrimSpace(args))

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file error: %w", err)
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
			printer.Error("json parse error", "line", line, "error", err)
			continue
		}

		source := entry.Source
		message := entry.Message

		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			if err := p.idx.IndexURL(source, message); err != nil {
				printer.Error("index URL failed", "url", source, "error", err)
				continue
			}
		} else {
			if !filepath.IsAbs(source) {
				source = filepath.Join(workDir, source)
			}
			if err := p.idx.IndexFile(source, message); err != nil {
				printer.Error("index file failed", "file", source, "error", err)
				continue
			}
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	return plugin.NopChatCommandResponse, nil
}
