package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/lmittmann/tint"
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

// CompactConfig controls the history-summarization behaviour of the chat agent.
type CompactConfig struct {
	// Threshold is the token count at which history summarization is triggered.
	// If zero or negative, defaults to 80 % of LLM.Context.
	Threshold int `koanf:"threshold"`

	// SaveTail is the number of most-recent messages to preserve unchanged
	// during summarization. If less than 1, all messages are passed to the
	// summarizer.
	SaveTail int `koanf:"save_tail"`
}

// Config is the root configuration structure.
type Config struct {
	LLM     LLMConfig     `koanf:"llm"`
	RAG     RAGConfig     `koanf:"rag"`
	Log     LogConfig     `koanf:"log"`
	Compact CompactConfig `koanf:"compact"`
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
// Output is written to stderr using tint for colourised, human-readable formatting.
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

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:   level,
		NoColor: !isTerminal(os.Stderr),
	})
	slog.SetDefault(slog.New(handler))
}

// isTerminal reports whether f is connected to a terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
