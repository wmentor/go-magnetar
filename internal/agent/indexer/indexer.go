package indexer

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/wmentor/go-magnetar/internal/chunk"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

var (
	extRegExp = regexp.MustCompile(`^\.(md|txt)$`)
)

// Indexer chunks content into overlapping segments and stores them in the RAG knowledge base.
type Indexer struct {
	cfg *config.Config
	rag *rag.RAGTools
	web *web.WebTools
}

// New creates a new Indexer instance.
func New(cfg *config.Config) (*Indexer, error) {
	ragTools, err := rag.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("indexer: failed to initialise RAG tools: %w", err)
	}

	webTools, err := web.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("indexer: failed to initialise web tools: %w", err)
	}

	return &Indexer{
		cfg: cfg,
		rag: ragTools,
		web: webTools,
	}, nil
}

// IndexFile indexes a single file into the RAG knowledge base.
func (idx *Indexer) IndexFile(filename string) error {
	slog.Info("indexer: indexing file", "file", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("indexer: failed to read file %q: %w", filename, err)
	}

	return idx.chunkAndSave(string(data))
}

// IndexURL fetches content from a URL, chunks it, and stores it in the RAG knowledge base.
func (idx *Indexer) IndexURL(rawURL string) error {
	slog.Info("indexer: indexing URL", "url", rawURL)

	content, err := idx.web.WebFetch(rawURL)
	if err != nil {
		return fmt.Errorf("indexer: failed to fetch URL %q: %w", rawURL, err)
	}

	return idx.chunkAndSave(content)
}

// chunkAndSave splits content into overlapping chunks and saves each one to RAG.
func (idx *Indexer) chunkAndSave(content string) error {
	cfg := chunk.Config{
		MaxSize: idx.cfg.RAG.Chunk.Size,
		Overlap: idx.cfg.RAG.Chunk.Overlap,
	}
	chunks := chunk.Split(content, cfg)
	saved := 0

	for i, c := range chunks {
		if idx.rag.RagSave(c) {
			saved++
		}
		slog.Debug("indexer: chunk saved", "chunk", i+1, "size", len(c))
	}

	slog.Info("indexer: done", "chunk", len(chunks), "saved", saved)
	return nil
}

// IndexDirectory indexes all .md and .txt files in the given directory tree.
func (idx *Indexer) IndexDirectory(dir string) error {
	slog.Info("indexer: indexing directory", "dir", dir)

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("indexer: walk error", "path", path, "err", err)
			return nil // continue walking
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !extRegExp.MatchString(ext) {
			return nil
		}

		if err := idx.IndexFile(path); err != nil {
			slog.Error("indexer: failed to index file", "file", path, "err", err)
		}

		return nil
	})
}
