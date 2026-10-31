package indexer

import (
	"context"
	"fmt"
	"io/fs"
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
	store rag.Store
	fetch *webfetch.Tool
}

// excludedDirNames are directories that should not be indexed.
var excludedDirNames = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"__pycache__":  {},
}

// plainTextExts are file extensions that are considered plain text.
var plainTextExts = map[string]struct{}{".md": {}, ".txt": {}}

// New creates a new Indexer instance.
func New(cfg *config.Config, root *os.Root, state *plugin.State) (*Indexer, error) {
	vecStore, err := store.NewStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("init RAG store error: %w", err)
	}

	fetchTool := webfetch.NewTool(cfg, state.Printer, state.RequesterFactory, state)

	return &Indexer{
		cfg:   cfg,
		store: vecStore,
		fetch: fetchTool,
	}, nil
}

// IndexFile indexes a single file into the RAG knowledge base.
func (idx *Indexer) IndexFile(filename string, msg string) error {
	_, err := idx.indexFileChunks(filename, msg)
	return err
}

// indexFile reads a single file, indexes it and returns the number of saved chunks.
func (idx *Indexer) indexFileChunks(filename, msg string) (int, error) {
	printer.Print(printer.IconFile, "indexer: indexing file", "file", filename)

	ext := strings.ToLower(filepath.Ext(filename))

	var data []byte
	var err error

	if reader, e := codec.GetCodec(ext); e == nil {
		text, err := reader.ReadFile(filename)
		if err != nil {
			return 0, fmt.Errorf("indexer: failed to read %s file %q: %w", ext, filename, err)
		}
		data = []byte(text)
	} else {
		data, err = os.ReadFile(filename)
		if err != nil {
			return 0, fmt.Errorf("indexer: failed to read file %q: %w", filename, err)
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

	_, err = idx.chunkAndSave(content, msg)
	return err
}

// chunkAndSave splits content into overlapping chunks and saves each one to RAG.
func (idx *Indexer) chunkAndSave(content string, msg string) (int, error) {
	cfg := chunk.Config{
		MaxSize: idx.cfg.Int("rag.chunk.size"),
		Overlap: idx.cfg.Int("rag.chunk.overlap"),
	}
	chunks := chunk.Split(content, cfg)
	saved := 0

	for i, c := range chunks {
		if idx.store.RagSave(c, msg, i+1) {
			saved++
		}
	}
	return saved, nil
}

// IndexDirectory indexes all indexable files in a directory into the RAG knowledge base.
func (idx *Indexer) IndexDirectory(root string, msg string) error {
	files, err := listIndexableFiles(root)
	if err != nil {
		return fmt.Errorf("indexer: failed to list files in %q: %w", root, err)
	}

	var (
		indexed  int
		chunks   int
		errFiles int
	)

	for _, file := range files {
		n, err := idx.indexFileChunks(file, msg)
		if err != nil {
			printer.Warn("indexer: failed to index file", "file", file, "err", err)
			errFiles++
			continue
		}
		chunks += n
		indexed++
	}

	if len(files) == 0 {
		printer.Warn("indexer: no indexable files found", "directory", root)
	} else {
		printer.Print(printer.IconFile, "indexer: directory indexed", "files", indexed, "chunks", chunks, "failed", errFiles)
	}
	return nil
}

// listIndexableFiles lists all indexable files in a directory.
func listIndexableFiles(path string) ([]string, error) {
	var found []string

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == path {
				return err
			}
			printer.Warn("indexer: skipping unreadable entry", "path", p, "err", err)
			return nil
		}

		if d.IsDir() {
			if _, ok := excludedDirNames[d.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}

		if !isIndexable(p) {
			return nil
		}

		found = append(found, p)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", path, err)
	}

	return found, nil
}

// isIndexable checks if a file is indexable based on its extension.
func isIndexable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := plainTextExts[ext]; ok {
		return true
	}

	if _, err := codec.GetCodec(ext); err == nil {
		return true
	}
	return false
}
