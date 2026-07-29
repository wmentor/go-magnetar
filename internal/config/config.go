package config

import (
	"fmt"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config is the root configuration structure.
type Config struct {
	cfg *koanf.Koanf
}

// defaults holds default config values that are applied before the
// YAML file is loaded, so the user doesn't need to set them.
var defaults = map[string]any{
	"rag.chunk.size":             512,       // runes; matches DefaultConfig in internal/chunk
	"rag.chunk.overlap":          64,        // runes; ~12.5% overlap is the RAG sweet-spot
	"rag.search.limit":           10,        // top-N results per query
	"rag.search.threshold":       0.40,      // minimum cosine-similarity score
	"rag.search.multi_query":     2,         // number of extra query reformulations
	"rag.search.dedup_threshold": 0.95,      // near-duplicate suppression threshold
	"llm.temperature":            0.5,       // LLM temperature for response generation
	"llm.top_p":                  0.95,      // LLM top_p for response generation
	"language":                   "english", // language for agent responses (default: english)
	"gitlab.base_url":            "",        // GitLab base URL (optional)
	"gitlab.api_key":             "",        // GitLab API key (optional)
	"github.base_url":            "",        // GitHub base URL (optional)
	"github.access_key":          "",        // GitHub access key (optional)
	"github.disable":             false,     // disable GitHub integration (default: false)
	"guard.disable":              false,     // disable guard agent for exec commands (default: false)
	"guard.ask":                  false,     // ask user for confirmation when guard blocks a command (default: false)
}

// Load reads and parses a YAML config file at the given path.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("config: failed to load defaults: %w", err)
	}

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("config: failed to load %q: %w", path, err)
	}

	return &Config{cfg: k}, nil
}

// String returns the string value for the given key.
func (c *Config) String(key string) string {
	return c.cfg.String(key)
}

// Bool returns the boolean value for the given key.
func (c *Config) Bool(key string) bool {
	return c.cfg.Bool(key)
}

// Int returns the integer value for the given key.
func (c *Config) Int(key string) int {
	return c.cfg.Int(key)
}

// Float64 returns the float64 value for the given key.
func (c *Config) Float64(key string) float64 {
	return c.cfg.Float64(key)
}

// isTerminal reports whether f is connected to a terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
