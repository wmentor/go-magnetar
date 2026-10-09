package plugin

import (
	"context"
)

type Skill struct {
	Name        string
	Description string
	Content     string
	Enable      bool
}

func (s *Skill) Clone() *Skill {
	return &Skill{
		Name:        s.Name,
		Description: s.Description,
		Content:     s.Content,
		Enable:      s.Enable,
	}
}

type SkillStore interface {
	List(ctx context.Context) ([]*Skill, error)
	Get(ctx context.Context, skillName string) (*Skill, error)
}
