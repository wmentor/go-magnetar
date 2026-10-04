package store

import (
	"fmt"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
	"github.com/wmentor/go-magnetar/internal/tools/rag/chromem"
	"github.com/wmentor/go-magnetar/internal/tools/rag/qdrant"
)

// NewStore creates a new Store instance based on configuration.
// Supports both qdrant and chromem stores.
// This function abstracts store selection to avoid circular dependencies.
func NewStore(cfg *config.Config) (rag.Store, error) {
	storeType := cfg.String("rag.store.type")
	if storeType == "" {
		storeType = "qdrant"
	}

	switch storeType {
	case "qdrant":
		return qdrant.NewStore(cfg)
	case "chromem":
		return chromem.NewStore(cfg)
	default:
		return nil, fmt.Errorf("rag: unknown store type %q", storeType)
	}
}
