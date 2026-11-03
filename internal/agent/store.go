package agent

import (
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

type Store struct {
	dirs   []string
	agents map[string]*plugin.Agent
}

func NewStore(dirs ...string) *Store {
	st := &Store{
		dirs:   slices.Clone(dirs),
		agents: make(map[string]*plugin.Agent),
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

		if strings.HasSuffix(d.Name(), ".md") {
			st.loadAgent(path)
		}
		return nil
	})
}

func (st *Store) loadAgent(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		printer.Debug("load agent %q error: %v", filename, err)
		return
	}

	texts := strings.SplitN(string(data), "---\n", 3)
	if len(texts) < 3 {
		printer.Debug("parse agent %q error: missing metadata section", filename)
		return
	}

	var metadata struct {
		Name           string   `yaml:"name"`
		Description    string   `yaml:"description"`
		AvailableTools []string `yaml:"available_tools"`
		Disable        bool     `yaml:"disable"`
		Subagents      []string `yaml:"subagents"`
	}

	if err := yaml.Unmarshal([]byte(texts[1]), &metadata); err != nil {
		printer.Debug("parse agent %q metadata error: %v", filename, err)
		return
	}

	if metadata.Name != "" {
		st.agents[metadata.Name] = &plugin.Agent{
			Name:           metadata.Name,
			Description:    metadata.Description,
			Prompt:         texts[2],
			AvailableTools: metadata.AvailableTools,
			Disable:        metadata.Disable,
			Subagents:      metadata.Subagents,
		}
	}
}

func (st *Store) Get(agentName string) (*plugin.Agent, error) {
	if agent, ok := st.agents[agentName]; ok {
		return agent.Copy(), nil
	}
	return nil, plugin.ErrAgentNotFound
}

func (st *Store) List() []*plugin.Agent {
	keys := maps.Keys(st.agents)
	sort.Strings(keys)
	ret := make([]*plugin.Agent, 0, len(keys))

	for _, k := range keys {
		ret = append(ret, st.agents[k].Copy())
	}

	return ret
}
