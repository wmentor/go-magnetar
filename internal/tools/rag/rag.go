package rag

import (
	"errors"

	"github.com/wmentor/go-magnetar/internal/config"
)

var (
	ErrInvalidConfig = errors.New("invalid value of config")
	ErrInvalidStore  = errors.New("invalid value of vector store")
)

// Store interface defines RAG operations.
type Store interface {
	Name() string
	RagSearch(query string) string
	RagSave(content string, prepend string, part int) bool
}

// Deprecated: Use the Tool struct with Exec method instead.
// RAGTools provides RAG operations as LLM tools.
type RAGTools struct {
	cfg   *config.Config
	store Store
}

// NewRAGTools creates a new RAGTools instance.
// Deprecated: Use the Tool struct with Exec method instead.
func NewRAGTools(cfg *config.Config, store Store) (*RAGTools, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}

	if store == nil || store == Store(nil) {
		return nil, ErrInvalidStore
	}

	return &RAGTools{
		cfg:   cfg,
		store: store,
	}, nil
}

// RagSave saves a text fragment to the store.
// Deprecated: Use the store directly or through the Tool interface.
func (r *RAGTools) RagSave(content string, prepend string, part int) bool {
	return r.store.RagSave(content, prepend, part)
}
