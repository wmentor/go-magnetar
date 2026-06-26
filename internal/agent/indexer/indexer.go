package indexer

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/wmentor/go-magnetar/internal/chunk"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web"
)

// Indexer chunks content into overlapping segments and stores them in the RAG knowledge base.
type Indexer struct {
	cfg *config.Config
	rag *rag.RAGTools
	web *web.WebTools
}

// New creates a new Indexer instance.
func New(cfg *config.Config, root *os.Root) (*Indexer, error) {
	ragTools, err := rag.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("indexer: failed to initialise RAG tools: %w", err)
	}

	webTools, err := web.New(cfg, root)
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
func (idx *Indexer) IndexFile(filename string, msg string) error {
	slog.Info("indexer: indexing file", "file", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("indexer: failed to read file %q: %w", filename, err)
	}

	return idx.chunkAndSave(string(data), msg)
}

// IndexURL fetches content from a URL, chunks it, and stores it in the RAG knowledge base.
func (idx *Indexer) IndexURL(rawURL string, msg string) error {
	slog.Info("indexer: indexing URL", "url", rawURL)

	content, err := idx.web.WebFetch(rawURL)
	if err != nil {
		return fmt.Errorf("indexer: failed to fetch URL %q: %w", rawURL, err)
	}

	return idx.chunkAndSave(content, msg)
}

// chunkAndSave splits content into overlapping chunks and saves each one to RAG.
func (idx *Indexer) chunkAndSave(content string, msg string) error {
	cfg := chunk.Config{
		MaxSize: idx.cfg.Int("rag.chunk.size"),
		Overlap: idx.cfg.Int("rag.chunk.overlap"),
	}
	chunks := chunk.Split(content, cfg)
	saved := 0

	for i, c := range chunks {
		if idx.rag.RagSave(c, msg) {
			saved++
		}
		slog.Debug("indexer: chunk saved", "chunk", i+1, "size", len(c))
	}

	slog.Info("indexer: done", "chunk", len(chunks), "saved", saved)
	return nil
}
