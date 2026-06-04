package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// LLMConfig holds settings for the main chat LLM.
type LLMConfig struct {
	BaseURL string `koanf:"base_url"`
	APIKey  string `koanf:"api_key"`
	Model   string `koanf:"model"`
	Context int    `koanf:"context"`
}

// EmbeddingConfig holds settings for the embedding model used in RAG.
type EmbeddingConfig struct {
	BaseURL    string `koanf:"base_url"`
	APIKey     string `koanf:"api_key"`
	Model      string `koanf:"model"`
	VectorSize int    `koanf:"vector_size"`
}

// QdrantConfig holds connection settings for Qdrant.
type QdrantConfig struct {
	ConnStr    string `koanf:"connstr"`
	Collection string `koanf:"collection"`
}

// RAGConfig combines embedding model and Qdrant settings.
type RAGConfig struct {
	LLM    EmbeddingConfig `koanf:"llm"`
	Qdrant QdrantConfig    `koanf:"qdrant"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `koanf:"level"` // debug | info | warn | error
}

// Config is the root configuration structure.
type Config struct {
	LLM LLMConfig `koanf:"llm"`
	RAG RAGConfig `koanf:"rag"`
	Log LogConfig `koanf:"log"`
}

// Load reads and parses a YAML config file at the given path.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("config: failed to load %q: %w", path, err)
	}

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal: %w", err)
	}

	return cfg, nil
}

// SetupLogger initialises the global slog logger based on the config level.
func SetupLogger(cfg *Config) {
	var level slog.Level

	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}
