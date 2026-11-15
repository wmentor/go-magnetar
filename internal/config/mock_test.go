package config_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
)

func TestNewMockEmpty(t *testing.T) {
	cfg := config.NewMock(nil)
	if cfg == nil {
		t.Fatal("NewMock(nil) returned nil")
	}

	if got := cfg.String("any.key"); got != "" {
		t.Errorf("String(\"any.key\") = %q, want \"\"", got)
	}
	if got := cfg.Int("any.key"); got != 0 {
		t.Errorf("Int(\"any.key\") = %d, want 0", got)
	}
	if got := cfg.Bool("any.key"); got != false {
		t.Errorf("Bool(\"any.key\") = %v, want false", got)
	}
	if got := cfg.Profile(); got != "" {
		t.Errorf("Profile() = %q, want \"\"", got)
	}
}

func TestNewMockWithValues(t *testing.T) {
	cfg := config.NewMock(map[string]any{
		"profile": "prod",
		"profiles": map[string]any{
			"prod": map[string]any{
				"llm": map[string]any{
					"model":       "claude-3-7-sonnet",
					"temperature": 0.5,
				},
			},
		},
		"rag.search.limit": 15,
		"guard.enable":     true,
	})

	if got := cfg.Profile(); got != "prod" {
		t.Errorf("Profile() = %q, want \"prod\"", got)
	}
	if got := cfg.String("profile"); got != "prod" {
		t.Errorf("String(\"profile\") = %q, want \"prod\"", got)
	}
	if got := cfg.ProfileParamString("llm.model"); got != "claude-3-7-sonnet" {
		t.Errorf("ProfileParamString(\"llm.model\") = %q, want \"claude-3-7-sonnet\"", got)
	}
	if got := cfg.ProfileParamFloat64("llm.temperature"); got != 0.5 {
		t.Errorf("ProfileParamFloat64(\"llm.temperature\") = %v, want 0.5", got)
	}
	if got := cfg.Int("rag.search.limit"); got != 15 {
		t.Errorf("Int(\"rag.search.limit\") = %d, want 15", got)
	}
	if got := cfg.Bool("guard.enable"); got != true {
		t.Errorf("Bool(\"guard.enable\") = %v, want true", got)
	}
}

func TestNewMockDefaultProfileInference(t *testing.T) {
	cfg := config.NewMock(map[string]any{
		"profiles": map[string]any{
			"default": map[string]any{
				"llm": map[string]any{
					"model": "gpt-4o",
				},
			},
		},
	})

	if got := cfg.Profile(); got != "default" {
		t.Errorf("Profile() = %q, want inferred \"default\"", got)
	}
	if got := cfg.ProfileParamString("llm.model"); got != "gpt-4o" {
		t.Errorf("ProfileParamString(\"llm.model\") = %q, want \"gpt-4o\"", got)
	}
}

func TestNewMockWithDefaults(t *testing.T) {
	cfg := config.NewMockWithDefaults(map[string]any{
		"compact.mode": "compact",
	})

	// Overridden default
	if got := cfg.String("compact.mode"); got != "compact" {
		t.Errorf("String(\"compact.mode\") = %q, want \"compact\"", got)
	}
	// Untouched standard defaults
	if got := cfg.Int("rag.chunk.size"); got != 2048 {
		t.Errorf("Int(\"rag.chunk.size\") = %d, want 2048", got)
	}
	if got := cfg.Float64("llm.temperature"); got != 0.9 {
		t.Errorf("Float64(\"llm.temperature\") = %v, want 0.9", got)
	}
}

func TestNewMockFromYAML(t *testing.T) {
	yamlStr := `
version: "1.0"
profile: test
profiles:
  test:
    llm:
      model: test-llm-model
      temperature: 0.7
      max_tokens: 4096
rag:
  search:
    limit: 25
`

	cfg, err := config.NewMockFromYAML(yamlStr)
	if err != nil {
		t.Fatalf("NewMockFromYAML failed: %v", err)
	}

	if got := cfg.Profile(); got != "test" {
		t.Errorf("Profile() = %q, want \"test\"", got)
	}
	if got := cfg.ProfileParamString("llm.model"); got != "test-llm-model" {
		t.Errorf("ProfileParamString(\"llm.model\") = %q, want \"test-llm-model\"", got)
	}
	if got := cfg.ProfileParamFloat64("llm.temperature"); got != 0.7 {
		t.Errorf("ProfileParamFloat64(\"llm.temperature\") = %v, want 0.7", got)
	}
	if got := cfg.ProfileParamInt("llm.max_tokens"); got != 4096 {
		t.Errorf("ProfileParamInt(\"llm.max_tokens\") = %d, want 4096", got)
	}
	if got := cfg.Int("rag.search.limit"); got != 25 {
		t.Errorf("Int(\"rag.search.limit\") = %d, want 25", got)
	}
}

func TestNewMockFromYAML_Invalid(t *testing.T) {
	_, err := config.NewMockFromYAML("invalid: yaml: : [unbalanced")
	if err == nil {
		t.Fatal("expected error on invalid YAML, got nil")
	}
}

func TestConfigSetAndSetProfileParam(t *testing.T) {
	cfg := config.NewMock(map[string]any{
		"profile": "staging",
		"profiles": map[string]any{
			"staging": map[string]any{
				"llm": map[string]any{
					"model": "gpt-3.5-turbo",
				},
			},
		},
	})

	if err := cfg.Set("custom.setting", "active"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if got := cfg.String("custom.setting"); got != "active" {
		t.Errorf("String(\"custom.setting\") = %q, want \"active\"", got)
	}

	// Update active profile via Set
	if err := cfg.Set("profile", "prod"); err != nil {
		t.Fatalf("Set(\"profile\") failed: %v", err)
	}
	if got := cfg.Profile(); got != "prod" {
		t.Errorf("Profile() = %q, want \"prod\"", got)
	}

	// Set parameter for current profile
	if err := cfg.SetProfileParam("llm.model", "gpt-4o-mini"); err != nil {
		t.Fatalf("SetProfileParam failed: %v", err)
	}
	if got := cfg.ProfileParamString("llm.model"); got != "gpt-4o-mini" {
		t.Errorf("ProfileParamString(\"llm.model\") = %q, want \"gpt-4o-mini\"", got)
	}
}
