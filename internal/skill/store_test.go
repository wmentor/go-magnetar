package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	t.Parallel()

	testDataDir := filepath.Join("testdata", "skill1")
	_, err := os.Stat(testDataDir)
	if os.IsNotExist(err) {
		t.Fatal("testdata/skill1 directory not found")
	}

	store := NewStore(testDataDir)

	ctx := context.Background()

	t.Run("List", func(t *testing.T) {
		skills, err := store.List(ctx)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}

		if len(skills) != 1 {
			t.Errorf("expected 1 skill, got %d", len(skills))
		}

		if skills[0].Name != "test-skill-1" {
			t.Errorf("expected skill name 'test-skill-1', got %q", skills[0].Name)
		}

		if skills[0].Description != "Test skill 1" {
			t.Errorf("expected description 'Test skill 1', got %q", skills[0].Description)
		}

		if skills[0].Content != "This is the content of test skill 1.\n" {
			t.Errorf("unexpected content: %q", skills[0].Content)
		}

		if !skills[0].Enable {
			t.Error("expected skill to be enabled")
		}

		skillPtr := skills[0]
		if skillPtr == nil {
			t.Fatal("skill pointer is nil")
		}

		cloned := skillPtr.Clone()
		if cloned == skillPtr {
			t.Error("List() returned same pointer instead of clone")
		}
	})

	t.Run("Get", func(t *testing.T) {
		skill, err := store.Get(ctx, "test-skill-1")
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}

		if skill.Name != "test-skill-1" {
			t.Errorf("expected skill name 'test-skill-1', got %q", skill.Name)
		}

		if skill.Enable != true {
			t.Error("expected skill to be enabled")
		}
	})

	t.Run("Get not found", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent-skill")
		if err != ErrSkillNotFound {
			t.Errorf("expected ErrSkillNotFound, got %v", err)
		}
	})

	t.Run("multiple directories", func(t *testing.T) {
		testDataDir2 := filepath.Join("testdata", "skill2")
		_, err := os.Stat(testDataDir2)
		if os.IsNotExist(err) {
			t.Fatal("testdata/skill2 directory not found")
		}

		store2 := NewStore(testDataDir, testDataDir2)

		skills, err := store2.List(ctx)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}

		if len(skills) != 2 {
			t.Errorf("expected 2 skills, got %d", len(skills))
		}

		names := make(map[string]bool)
		for _, s := range skills {
			names[s.Name] = true
		}

		if !names["test-skill-1"] {
			t.Error("expected test-skill-1 in list")
		}

		if !names["test-skill-2"] {
			t.Error("expected test-skill-2 in list")
		}
	})

	t.Run("hidden directories skipped", func(t *testing.T) {
		hiddenDir := filepath.Join("testdata", ".hidden")
		_, err := os.Stat(hiddenDir)
		if os.IsNotExist(err) {
			t.Fatal("testdata/.hidden directory not found")
		}

		store3 := NewStore(hiddenDir)

		skills, err := store3.List(ctx)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}

		if len(skills) != 0 {
			t.Errorf("expected 0 skills (hidden dir should be skipped), got %d", len(skills))
		}
	})
}
