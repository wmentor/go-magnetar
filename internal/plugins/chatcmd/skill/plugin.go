package skillplugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.skill", &Plugin{})
}

// Plugin registers the /skill chat command.
type Plugin struct{}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "skill",
		Aliases: []string{"sk"},
		Help:    "List available skills or load/activate a skill",
		Execute: func(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			args := params.Args
			if args == "" {
				return &plugin.ChatCommandResponse{
					Stdout: listSkills(state.SkillStore),
				}, nil
			}

			parts := strings.SplitN(args, " ", 2)
			skillName := strings.TrimSpace(parts[0])
			optionalText := ""
			if len(parts) == 2 {
				optionalText = strings.TrimSpace(parts[1])
			}

			skill, err := state.SkillStore.Get(ctx, skillName)
			if err != nil {
				return nil, fmt.Errorf("failed to load skill '%s': %w", skillName, err)
			}

			if !skill.Enable {
				return nil, fmt.Errorf("skill '%s' is not enabled", skillName)
			}

			return &plugin.ChatCommandResponse{
				Content: makeApplySkillMessage(skill, optionalText),
				Stdout:  fmt.Sprintf("Loaded skill: %s\n\n", skillName),
			}, nil
		},
	})
	return nil
}

func listSkills(skillStore plugin.SkillStore) string {
	skills, err := skillStore.List(context.Background())
	if err != nil {
		return fmt.Sprintf("failed to list skills: %v\n", err)
	}

	var sb strings.Builder
	sb.WriteString("Available skills:\n")

	for _, skill := range skills {
		if !skill.Enable {
			continue
		}
		if skill.Description != "" {
			fmt.Fprintf(&sb, "  * %s - %s\n", skill.Name, skill.Description)
		} else {
			fmt.Fprintf(&sb, "  * %s\n", skill.Name)
		}
	}

	return sb.String()
}

func makeApplySkillMessage(skill *plugin.Skill, optionalText string) string {
	var sb strings.Builder

	sb.WriteString("# SKILL: ")
	sb.WriteString(skill.Name)
	sb.WriteString("\n\n")
	sb.WriteString(skill.Content)

	sb.WriteString("\n\nIMPORTANT: to solve the next problem, use the skill `")
	sb.WriteString(skill.Name)
	sb.WriteString("`\n")

	if optionalText != "" {
		sb.WriteString("# Problem:\n\n")
		sb.WriteString(optionalText)
		sb.WriteString("\n")
	}

	return sb.String()
}
