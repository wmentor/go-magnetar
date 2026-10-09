package skill

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/exp/maps"

	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

var (
	ErrSkillNotFound = errors.New("skill not found")

	_ plugin.SkillStore = (*Store)(nil)
)

type Store struct {
	dirs   []string
	skills map[string]*plugin.Skill
}

func NewStore(dirs ...string) *Store {
	st := &Store{
		dirs:   slices.Clone(dirs),
		skills: make(map[string]*plugin.Skill),
	}

	for _, d := range st.dirs {
		st.readDir(d)
	}

	return st
}

func (st *Store) readDir(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() == "SKILL.md" {
			st.loadSkill(path)
		}
		return nil
	})
}

func (st *Store) loadSkill(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		printer.Debug("load skill %q error: %v", filename, err)
		return
	}

	texts := strings.SplitN(string(data), "---\n", 3)
	if len(texts) < 3 {
		printer.Debug("parse skill %q error: %v", filename, err)
		return
	}

	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Disable     bool   `yaml:"disable"`
	}

	if err := yaml.Unmarshal([]byte(texts[1]), &metadata); err != nil {
		printer.Debug("parse skill %q metadata error: %v", filename, err)
	}

	if metadata.Name != "" {
		st.skills[metadata.Name] = &plugin.Skill{
			Name:        metadata.Name,
			Description: metadata.Description,
			Content:     texts[2],
			Enable:      !metadata.Disable,
		}
	}
}

func (st *Store) Get(ctx context.Context, skillName string) (*plugin.Skill, error) {
	if skill, ok := st.skills[skillName]; ok {
		return skill.Clone(), nil
	}
	return nil, ErrSkillNotFound
}

func (st *Store) List(ctx context.Context) ([]*plugin.Skill, error) {
	keys := maps.Keys(st.skills)
	sort.Strings(keys)
	ret := make([]*plugin.Skill, 0, len(keys))

	for _, k := range keys {
		ret = append(ret, st.skills[k].Clone())
	}

	return ret, nil
}
