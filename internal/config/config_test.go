package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSetProfileUpdatesProfileKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	testConfig := `version: "1.0"
profile: work

profiles:
  work:
    llm:
      model: work-model
  home:
    llm:
      model: home-model
`
	if err := os.WriteFile(path, []byte(testConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := cfg.SetProfile("home"); err != nil {
		t.Fatal(err)
	}
	if got := cfg.String("profile"); got != "home" {
		t.Errorf("String(\"profile\") = %q after SetProfile(\"home\"), want \"home\"", got)
	}
	if got := cfg.Profile(); got != "home" {
		t.Errorf("Profile() = %q, want \"home\"", got)
	}
	if got := cfg.ProfileParamString("llm.model"); got != "home-model" {
		t.Errorf("ProfileParamString(\"llm.model\") = %q, want \"home-model\"", got)
	}

	// an unknown profile changes nothing
	if err := cfg.SetProfile("missing"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("SetProfile(\"missing\") error = %v, want ErrProfileNotFound", err)
	}
	if got := cfg.String("profile"); got != "home" {
		t.Errorf("String(\"profile\") = %q after a failed SetProfile, want \"home\"", got)
	}
}

func TestConfigCopy(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	testConfig := `version: "1.0"
profile: work

profiles:
  work:
    llm:
      model: work-model
      temperature: 0.9
    rag:
      search:
        limit: 5
  home:
    llm:
      model: home-model
      temperature: 0.7
`
	if err := os.WriteFile(path, []byte(testConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	// Set profile to work
	if err := cfg.SetProfile("work"); err != nil {
		t.Fatal(err)
	}

	// Create a copy
	cfgCopy := cfg.Copy()

	// Verify copy has same profile
	if got := cfgCopy.Profile(); got != "work" {
		t.Errorf("Copy().Profile() = %q, want \"work\"", got)
	}

	// Verify copy has same values
	if got := cfgCopy.String("profile"); got != "work" {
		t.Errorf("Copy().String(\"profile\") = %q, want \"work\"", got)
	}

	if got := cfgCopy.ProfileParamString("llm.model"); got != "work-model" {
		t.Errorf("Copy().ProfileParamString(\"llm.model\") = %q, want \"work-model\"", got)
	}

	if got := cfgCopy.ProfileParamFloat64("llm.temperature"); got != 0.9 {
		t.Errorf("Copy().ProfileParamFloat64(\"llm.temperature\") = %v, want 0.9", got)
	}

	// Verify copy is independent - modify original
	if err := cfg.SetProfile("home"); err != nil {
		t.Fatal(err)
	}

	// Copy should still have work profile
	if got := cfgCopy.Profile(); got != "work" {
		t.Errorf("After modifying original, Copy().Profile() = %q, want \"work\"", got)
	}

	// Original should have home profile
	if got := cfg.Profile(); got != "home" {
		t.Errorf("After SetProfile(\"home\"), Profile() = %q, want \"home\"", got)
	}
}
