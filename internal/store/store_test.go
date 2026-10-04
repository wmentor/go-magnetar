package store

import (
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
)

func TestNewStore(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join("testdata", "config.yml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	storeType := cfg.String("rag.store.type")
	if storeType != "chromem" {
		t.Errorf("Expected store type 'chromem', got '%s'", storeType)
	}

	db, err := NewStore(cfg)
	if err != nil {
		t.Errorf("NewStore() error = %v", err)
		return
	}

	// Verify store.Name() returns correct value
	name := db.Name()
	if name != "chromem" {
		t.Errorf("store.Name() = %q, want %q", name, "chromem")
	}
}
