package chromem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/philippgille/chromem-go"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

// Store implements rag.Store interface using chromem-go storage.
type Store struct {
	cfg         *config.Config
	db          *chromem.DB
	collection  *chromem.Collection
	embedClient *openai.Client
}

// NewStore creates a new chromem store instance.
// If rag.chromem.data_dir is empty, creates an in-memory store.
// Otherwise, creates a persistent store at the specified directory.
func NewStore(cfg *config.Config) (*Store, error) {
	dataDir := cfg.String("rag.chromem.data_dir")

	var db *chromem.DB
	var err error

	if dataDir == "" {
		// In-memory store
		db = chromem.NewDB()
	} else {
		// Persistent store
		homeDir, _ := os.UserHomeDir()
		dataDir = strings.Replace(dataDir, "~", homeDir, 1)

		dbPath := filepath.Join(dataDir, "chromem")
		if err := os.MkdirAll(dbPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create chromem data directory: %w", err)
		}

		db, err = chromem.NewPersistentDB(dbPath, true)
		if err != nil {
			return nil, fmt.Errorf("failed to create chromem persistent DB: %w", err)
		}
	}

	// Create or get collection
	collectionName := cfg.String("rag.chromem.collection")
	if collectionName == "" {
		collectionName = "documents"
	}

	collection, err := db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create chromem collection: %w", err)
	}

	// Build OpenAI embedding client (reusing rag.Embed client)
	embedCfg := openai.DefaultConfig(cfg.String("rag.llm.api_key"))
	embedCfg.BaseURL = cfg.String("rag.llm.base_url")
	embedClient := openai.NewClientWithConfig(embedCfg)

	return &Store{
		cfg:         cfg,
		db:          db,
		collection:  collection,
		embedClient: embedClient,
	}, nil
}

func (s *Store) Name() string {
	return "chromem"
}

// RagSearch searches the knowledge base and returns relevant text fragments.
func (s *Store) RagSearch(query string) string {
	printer.ToolCall(printer.IconSearch, "rag_search", "query", query)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get search parameters
	limit := s.cfg.Int("rag.search.limit")
	if limit <= 0 {
		limit = 10
	}

	threshold := float32(s.cfg.Float64("rag.search.threshold"))
	if threshold <= 0 {
		threshold = 0.40
	}

	// Get query embedding
	queryVector, err := rag.Embed(ctx, s.embedClient, s.cfg.String("rag.llm.model"), query)
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_search: embedding failed", "err", err)
		return ""
	}

	// Limit nResults to number of documents in collection
	docCount := s.collection.Count()
	if limit > docCount {
		limit = docCount
	}
	if limit <= 0 {
		printer.ToolCall(printer.IconSearch, "rag_search: no results", "query", query)
		return ""
	}

	// Query the collection
	results, err := s.collection.QueryEmbedding(ctx, queryVector, limit, nil, nil)
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_search: query failed", "err", err)
		return ""
	}

	// Filter by threshold and format results
	filtered := make([]chromem.Result, 0, len(results))
	for _, res := range results {
		if res.Similarity >= threshold {
			filtered = append(filtered, res)
		}
	}

	if len(filtered) == 0 {
		printer.ToolCall(printer.IconSearch, "rag_search: no results", "query", query)
		return ""
	}

	// Format output
	parts := make([]string, 0, len(filtered))
	for _, res := range filtered {
		parts = append(parts, res.Content)
	}

	printer.ToolCall(printer.IconSearch, "rag_search: done", "query", query,
		"results", len(filtered))
	return strings.Join(parts, "\n\n---\n\n")
}

// RagSave saves a text fragment to chromem. Returns true on success.
func (s *Store) RagSave(content string, prepend string, part int) bool {
	if prepend != "" {
		content = prepend + "\n" + content
	}

	id := rag.ContentUUID(content)

	printer.ToolCall(printer.IconSave, "rag_save", "id", id, "size", len(content), "part", part)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get embedding
	vector, err := rag.Embed(ctx, s.embedClient, s.cfg.String("rag.llm.model"), content)
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_save: embedding failed", "err", err)
		return false
	}

	// Add document to collection
	doc := chromem.Document{
		ID:        id,
		Content:   content,
		Metadata:  map[string]string{"part": fmt.Sprintf("%d", part)},
		Embedding: vector,
	}

	err = s.collection.AddDocument(ctx, doc)
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_save: failed to add document", "id", id, "err", err)
		return false
	}

	return true
}
