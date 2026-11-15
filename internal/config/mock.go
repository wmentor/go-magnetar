package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// NewMock creates an in-memory Config for unit tests with the provided key-value pairs.
// Keys can be flat dot-delimited (e.g. "profiles.default.llm.model") or nested maps.
// If "profile" is specified, it is set as the active profile.
func NewMock(values map[string]any) *Config {
	k := koanf.New(".")
	if len(values) > 0 {
		_ = k.Load(confmap.Provider(values, "."), nil)
	}

	profile := k.String("profile")
	if profile == "" {
		for _, p := range k.MapKeys("profiles") {
			if p == "default" {
				profile = "default"
				break
			}
		}
	}

	return &Config{
		cfg:       k,
		profile:   profile,
		configDir: "",
	}
}

// NewMockWithDefaults creates an in-memory Config preloaded with the standard
// default configuration values, overridden by the provided key-value pairs.
func NewMockWithDefaults(values map[string]any) *Config {
	k := koanf.New(".")
	_ = k.Load(confmap.Provider(defaultValues(), "."), nil)
	if len(values) > 0 {
		_ = k.Load(confmap.Provider(values, "."), nil)
	}

	profile := k.String("profile")
	if profile == "" {
		for _, p := range k.MapKeys("profiles") {
			if p == "default" {
				profile = "default"
				break
			}
		}
	}

	return &Config{
		cfg:       k,
		profile:   profile,
		configDir: "",
	}
}

// NewMockFromYAML creates an in-memory Config parsed from a YAML string without disk I/O.
func NewMockFromYAML(yamlContent string) (*Config, error) {
	k := koanf.New(".")
	parser := yaml.Parser()
	data, err := parser.Unmarshal([]byte(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("config: failed to parse YAML: %w", err)
	}
	if err := k.Load(confmap.Provider(data, "."), nil); err != nil {
		return nil, fmt.Errorf("config: failed to load confmap: %w", err)
	}

	profile := k.String("profile")
	if profile == "" {
		for _, p := range k.MapKeys("profiles") {
			if p == "default" {
				profile = "default"
				break
			}
		}
	}

	return &Config{
		cfg:       k,
		profile:   profile,
		configDir: "",
	}, nil
}

// Set dynamically sets or updates a configuration key in-memory.
// If key is "profile", the active profile is updated as well.
func (c *Config) Set(key string, val any) error {
	if err := c.cfg.Set(key, val); err != nil {
		return err
	}
	if key == "profile" {
		c.profile = fmt.Sprintf("%v", val)
	}
	return nil
}

// SetProfileParam sets a parameter specifically under the active profile
// (equivalent to Set("profiles.<profile>.<key>", val)).
func (c *Config) SetProfileParam(key string, val any) error {
	return c.Set(c.makeProfileKey(key), val)
}
