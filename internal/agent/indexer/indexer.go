package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wmentor/go-magnetar/internal/chunk"
	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/store"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/web/webfetch"
)

// Indexer chunks content into overlapping segments and stores them in the RAG knowledge base.
type Indexer struct {
	cfg   *config.Config
	rag   *rag.RAGTools
	fetch *webfetch.Tool
}

// New creates a new Indexer instance.
func New(cfg *config.Config, root *os.Root, state *plugin.State) (*Indexer, error) {
	vecStore, err := store.NewStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("init RAG store error: %w", err)
	}

	ragTools, err := rag.New(cfg, vecStore)
	if err != nil {
		return nil, fmt.Errorf("indexer: failed to initialise RAG tools: %w", err)
	}

	fetchTool := webfetch.NewTool(cfg, state.Printer, state.RequesterFactory, state)

	return &Indexer{
		cfg:   cfg,
		rag:   ragTools,
		fetch: fetchTool,
	}, nil
}

// IndexFile indexes a single file into the RAG knowledge base.
func (idx *Indexer) IndexFile(filename string, msg string) error {
	printer.Print(printer.IconFile, "indexer: indexing file", "file", filename)

	ext := strings.ToLower(filepath.Ext(filename))

	var data []byte
	var err error

	if reader, e := codec.GetCodec(ext); e == nil {
		text, err := reader.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("indexer: failed to read %s file %q: %w", ext, filename, err)
		}
		data = []byte(text)
	} else {
		data, err = os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("indexer: failed to read file %q: %w", filename, err)
		}
	}

	return idx.chunkAndSave(string(data), msg)
}

// IndexURL fetches content from a URL, chunks it, and stores it in the RAG knowledge base.
func (idx *Indexer) IndexURL(rawURL string, msg string) error {
	printer.Print(printer.IconURL, "indexer: indexing URL", "url", rawURL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	content, err := idx.fetch.WebFetchEngine(ctx, rawURL, false)
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
		if idx.rag.RagSave(c, msg, i+1) {
			saved++
		}
	}
	return nil
}
