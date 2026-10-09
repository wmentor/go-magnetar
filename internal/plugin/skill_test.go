package plugin

import (
	"testing"
)

func TestSkill_Clone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		skill *Skill
	}{
		{
			"empty skill",
			&Skill{},
		},
		{
			"skill with all fields",
			&Skill{
				Name:        "test-skill",
				Description: "test description",
				Content:     "test content",
				Enable:      true,
			},
		},
		{
			"skill with empty content",
			&Skill{
				Name:        "empty-content-skill",
				Description: "description with empty content",
				Enable:      false,
			},
		},
		{
			"skill with disabled flag",
			&Skill{
				Name:   "disabled-skill",
				Enable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := tt.skill.Clone()

			if cloned == nil {
				t.Fatal("Clone() returned nil")
			}

			if cloned.Name != tt.skill.Name {
				t.Errorf("Name: got %q, want %q", cloned.Name, tt.skill.Name)
			}

			if cloned.Description != tt.skill.Description {
				t.Errorf("Description: got %q, want %q", cloned.Description, tt.skill.Description)
			}

			if cloned.Content != tt.skill.Content {
				t.Errorf("Content: got %q, want %q", cloned.Content, tt.skill.Content)
			}

			if cloned.Enable != tt.skill.Enable {
				t.Errorf("Enable: got %v, want %v", cloned.Enable, tt.skill.Enable)
			}

			if cloned == tt.skill {
				t.Error("Clone() returned same pointer")
			}
		})
	}
}
