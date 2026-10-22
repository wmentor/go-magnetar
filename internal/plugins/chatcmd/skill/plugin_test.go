package skillplugin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/response"
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
			answer := listSkills(store)
			assert.Equal(t, tc.expected, answer)
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
			got := MakeApplySkillMessage(tt.skill, tt.optionalText)
			assert.Equal(t, tt.want, got)
		})
	}
}

// mockHub implements Hub interface for testing
type mockHub struct {
	RegisteredCommands []plugin.ChatCommand
}

func (m *mockHub) RegisterTool(tool plugin.LLMTool) {
	// no-op
}

func (m *mockHub) RegisterChatCommand(cmd plugin.ChatCommand) {
	m.RegisteredCommands = append(m.RegisteredCommands, cmd)
}

func (m *mockHub) RegisterPreprocessor(fn plugin.PreprocessorFunc) {
	// no-op
}

func (m *mockHub) Go(f func(ctx context.Context)) {
	// no-op
}

func (m *mockHub) Stop() {
	// no-op
}

// TestExecute tests the Execute function of the skill command
func TestExecute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupStore  func() plugin.SkillStore
		args        string
		wantErr     bool
		wantStdout  string
		wantContent string
	}{
		{
			name: "no args - list skills",
			setupStore: func() plugin.SkillStore {
				return &mockSkillStore{
					skills: []*plugin.Skill{
						{
							Name:        "coding",
							Description: "Programming assistant",
							Content:     "You are a coding expert...",
							Enable:      true,
						},
					},
				}
			},
			args:        "",
			wantErr:     false,
			wantStdout:  "Available skills:\n  * coding - Programming assistant\n",
			wantContent: "",
		},
		{
			name: "enabled skill with content",
			setupStore: func() plugin.SkillStore {
				return &mockSkillStore{
					skills: []*plugin.Skill{
						{
							Name:    "coding",
							Content: "You are a coding expert...",
							Enable:  true,
						},
					},
				}
			},
			args:        "coding",
			wantErr:     false,
			wantStdout:  "Loaded skill: coding\n\n",
			wantContent: "# SKILL: coding\n\nYou are a coding expert...\n\nIMPORTANT: to solve the next problem, use the skill `coding`\n",
		},
		{
			name: "enabled skill with optional text",
			setupStore: func() plugin.SkillStore {
				return &mockSkillStore{
					skills: []*plugin.Skill{
						{
							Name:    "coding",
							Content: "You are a coding expert...",
							Enable:  true,
						},
					},
				}
			},
			args:        "coding Write a haiku about Go",
			wantErr:     false,
			wantStdout:  "Loaded skill: coding\n\n",
			wantContent: "# SKILL: coding\n\nYou are a coding expert...\n\nIMPORTANT: to solve the next problem, use the skill `coding`\n# Problem:\n\nWrite a haiku about Go\n",
		},
		{
			name: "disabled skill",
			setupStore: func() plugin.SkillStore {
				return &mockSkillStore{
					skills: []*plugin.Skill{
						{
							Name:    "writing",
							Content: "You are a writing expert...",
							Enable:  false,
						},
					},
				}
			},
			args:        "writing",
			wantErr:     true,
			wantStdout:  "",
			wantContent: "",
		},
		{
			name: "non-existent skill",
			setupStore: func() plugin.SkillStore {
				return &mockSkillStore{
					skills: []*plugin.Skill{},
				}
			},
			args:        "nonexistent",
			wantErr:     true,
			wantStdout:  "",
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := tt.setupStore()
			responseStore := response.New(plugin.DefaultResponseStoreSize)
			state := &plugin.State{
				SkillStore:    store,
				ResponseStore: responseStore,
			}

			// Create plugin and register command
			p := &Plugin{}
			hub := &mockHub{}
			err := p.Init(state, hub)
			assert.NoError(t, err)

			// Get the registered command
			commands := hub.RegisteredCommands
			assert.Len(t, commands, 1)
			skillCommand := commands[0]

			input := &plugin.ChatCommandInput{
				Args: tt.args,
			}

			ctx := context.Background()
			response, err := skillCommand.Execute(ctx, input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if response != nil {
					assert.Equal(t, tt.wantStdout, response.Stdout)
					assert.Equal(t, tt.wantContent, response.Content)
				}
			}
		})
	}
}
