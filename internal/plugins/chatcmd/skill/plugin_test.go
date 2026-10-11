package skillplugin

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

type mockSkillStore struct {
	skills []*plugin.Skill
}

func (m *mockSkillStore) List(ctx context.Context) ([]*plugin.Skill, error) {
	return m.skills, nil
}

func (m *mockSkillStore) Get(ctx context.Context, skillName string) (*plugin.Skill, error) {
	for _, s := range m.skills {
		if s.Name == skillName {
			return s.Clone(), nil
		}
	}
	return nil, errors.New("skill not found")
}

func TestListSkills(t *testing.T) {
	testCases := []struct {
		name     string
		skills   []*plugin.Skill
		expected string
	}{
		{
			name:   "no skills",
			skills: []*plugin.Skill{},
			expected: `Available skills:
`,
		},
		{
			name: "one enabled skill",
			skills: []*plugin.Skill{
				{
					Name:        "coding",
					Description: "Programming assistant",
					Content:     "You are a coding expert...",
					Enable:      true,
				},
			},
			expected: `Available skills:
  * coding - Programming assistant
`,
		},
		{
			name: "mix of enabled and disabled skills",
			skills: []*plugin.Skill{
				{
					Name:        "coding",
					Description: "Programming assistant",
					Content:     "You are a coding expert...",
					Enable:      true,
				},
				{
					Name:        "disabled-skill",
					Description: "This is disabled",
					Content:     "Disabled content",
					Enable:      false,
				},
				{
					Name:        "writing",
					Description: "Writing assistant",
					Content:     "You are a writing expert...",
					Enable:      true,
				},
			},
			expected: `Available skills:
  * coding - Programming assistant
  * writing - Writing assistant
`,
		},
		{
			name: "skill without description",
			skills: []*plugin.Skill{
				{
					Name:        "coding",
					Description: "",
					Content:     "You are a coding expert...",
					Enable:      true,
				},
			},
			expected: `Available skills:
  * coding
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := &mockSkillStore{skills: tc.skills}
			var buf bytes.Buffer
			listSkills(&buf, store)
			assert.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestMakeApplySkillMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		skill        *plugin.Skill
		optionalText string
		want         string
	}{
		{
			name: "skill with content and no optional text",
			skill: &plugin.Skill{
				Name:    "coding",
				Content: "You are a coding expert...",
			},
			optionalText: "",
			want:         "# SKILL: coding\n\nYou are a coding expert...\n\nIMPORTANT: to solve the next problem, use the skill `coding`\n",
		},
		{
			name: "skill with content and optional text",
			skill: &plugin.Skill{
				Name:    "writing",
				Content: "You are a writing expert...",
			},
			optionalText: "Write a haiku about Go",
			want:         "# SKILL: writing\n\nYou are a writing expert...\n\nIMPORTANT: to solve the next problem, use the skill `writing`\n# Problem:\n\nWrite a haiku about Go\n",
		},
		{
			name: "skill with empty content and no optional text",
			skill: &plugin.Skill{
				Name:    "empty-skill",
				Content: "",
			},
			optionalText: "",
			want:         "# SKILL: empty-skill\n\n\n\nIMPORTANT: to solve the next problem, use the skill `empty-skill`\n",
		},
		{
			name: "skill with empty content and optional text",
			skill: &plugin.Skill{
				Name:    "empty-skill-with-text",
				Content: "",
			},
			optionalText: "Solve this problem",
			want:         "# SKILL: empty-skill-with-text\n\n\n\nIMPORTANT: to solve the next problem, use the skill `empty-skill-with-text`\n# Problem:\n\nSolve this problem\n",
		},
		{
			name: "skill with multiline content and optional text",
			skill: &plugin.Skill{
				Name:    "multiline-skill",
				Content: "Line 1\nLine 2\nLine 3",
			},
			optionalText: "Task with multiple\nlines",
			want:         "# SKILL: multiline-skill\n\nLine 1\nLine 2\nLine 3\n\nIMPORTANT: to solve the next problem, use the skill `multiline-skill`\n# Problem:\n\nTask with multiple\nlines\n",
		},
		{
			name: "skill with special characters",
			skill: &plugin.Skill{
				Name:    "special-skill",
				Content: "Content with `code` and **bold**",
			},
			optionalText: "Problem with $pecial chars",
			want:         "# SKILL: special-skill\n\nContent with `code` and **bold**\n\nIMPORTANT: to solve the next problem, use the skill `special-skill`\n# Problem:\n\nProblem with $pecial chars\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := makeApplySkillMessage(tt.skill, tt.optionalText)
			assert.Equal(t, tt.want, got)
		})
	}
}
