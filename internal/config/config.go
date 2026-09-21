package config

import (
	"fmt"
	"os/user"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	currentVersion = "1.0"
)

// Config is the root configuration structure.
type Config struct {
	cfg       *koanf.Koanf
	profile   string
	configDir string
}

// defaults holds default config values that are applied before the
// YAML file is loaded, so the user doesn't need to set them.
var defaults = map[string]any{
	"rag.chunk.size":             2048,            // runes; matches DefaultConfig in internal/chunk
	"rag.chunk.overlap":          256,             // runes; ~12.5% overlap is the RAG sweet-spot
	"rag.search.limit":           10,              // top-N results per query
	"rag.search.threshold":       0.40,            // minimum cosine-similarity score
	"rag.search.multi_query":     2,               // number of extra query reformulations
	"rag.search.dedup_threshold": 0.95,            // near-duplicate suppression threshold
	"llm.temperature":            0.9,             // LLM temperature for response generation
	"llm.top_p":                  0.95,            // LLM top_p for response generation
	"llm.reasoning_effort":       "high",          // LLM reasoning effort: low, medium, or high
	"language":                   "english",       // language for agent responses (default: english)
	"gitlab.base_url":            "",              // GitLab base URL (optional)
	"gitlab.api_key":             "",              // GitLab API key (optional)
	"github.base_url":            "",              // GitHub base URL (optional)
	"github.access_key":          "",              // GitHub access key (optional)
	"github.enable":              false,           // enable GitHub integration (default: false)
	"guard.enable":               false,           // enable guard agent for exec commands (default: false)
	"guard.ask":                  false,           // ask user for confirmation when guard blocks a command (default: false)
	"ssh.enable":                 false,           // enable SSH execution (default: false)
	"ssh.user":                   "",              // SSH username (optional, defaults to current user)
	"ssh.key":                    "~/.ssh/id_rsa", // SSH private key path (optional)
	"ssh.password":               "",              // SSH password (optional)
	"ssh.use_ssh_agent":          false,           // use ssh-agent for authentication (default: false)
	"ssh.remote_dir":             "",              // remote working directory (optional, default: $HOME)
	"ssh.timeout":                120,             // SSH command timeout in seconds (default: 120)
}

// Load reads and parses a YAML config file at the given path.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if currentUser, err := user.Current(); err == nil {
		defaults["ssh.user"] = currentUser
	}

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("config: failed to load defaults: %w", err)
	}

	// Load main config file (overrides defaults)
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("config: failed to load %q: %w", path, err)
	}

	if ver := k.String("version"); ver != currentVersion {
		return nil, fmt.Errorf("config: unsupported config version %q", ver)
	}

	// Load include files (override main values)
	includes := k.Strings("include")
	for _, includePath := range includes {
		if includePath == "" {
			continue
		}
		// Resolve include path relative to main config directory
		includeAbsPath, err := filepath.Abs(filepath.Join(filepath.Dir(path), includePath))
		if err != nil {
			return nil, fmt.Errorf("config: failed to resolve include path %q: %w", includePath, err)
		}

		if err := k.Load(file.Provider(includeAbsPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("config: failed to load include %q: %w", includeAbsPath, err)
		}
	}

	configDir := filepath.Dir(path)

	return &Config{
		cfg:       k,
		profile:   k.String("profile"),
		configDir: configDir,
	}, nil
}

// String returns the string value for the given key.
func (c *Config) String(key string) string {
	return ResolveEnvVars(c.cfg.String(key), c.configDir)
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

// String returns the string value for the given key.
func (c *Config) ProfileParamString(key string) string {
	return ResolveEnvVars(c.cfg.String(c.makeProfileKey(key)), c.configDir)
}

// Bool returns the boolean value for the given key.
func (c *Config) ProfileParamBool(key string) bool {
	return c.cfg.Bool(c.makeProfileKey(key))
}

// Int returns the integer value for the given key.
func (c *Config) ProfileParamInt(key string) int {
	return c.cfg.Int(c.makeProfileKey(key))
}

// Float64 returns the float64 value for the given key.
func (c *Config) ProfileParamFloat64(key string) float64 {
	return c.cfg.Float64(c.makeProfileKey(key))
}

// SetProfile sets the profile name to use for parameter lookup.
func (c *Config) SetProfile(profile string) {
	c.profile = profile
}

func (c *Config) makeProfileKey(key string) string {
	if c.profile == "" {
		return key
	}
	return fmt.Sprintf("profiles.%s.%s", c.profile, key)
}
