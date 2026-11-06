package agent

import (
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
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Profile     string   `yaml:"profile"`
		Tools       []string `yaml:"tools"`
		Disable     bool     `yaml:"disable"`
		Subagents   []string `yaml:"subagents"`
	}

	if err := yaml.Unmarshal([]byte(texts[1]), &metadata); err != nil {
		printer.Debug("parse agent %q metadata error: %v", filename, err)
		return
	}

	if metadata.Name != "" {
		st.agents[metadata.Name] = &plugin.Agent{
			Name:           metadata.Name,
			Description:    metadata.Description,
			Profile:        metadata.Profile,
			Prompt:         texts[2],
			AvailableTools: metadata.Tools,
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

// Filter creates a copy of the store containing only the specified agents.
// Returns an error if any of the specified agent names do not exist in the store.
func (st *Store) Filter(agents []string) (plugin.AgentStore, error) {
	if st == nil {
		return nil, errors.New("store is nil")
	}

	// Create a new store with the same directories
	newStore := &Store{
		dirs:   slices.Clone(st.dirs),
		agents: make(map[string]*plugin.Agent),
	}

	// Track which agents were not found
	notFound := make([]string, 0)

	for _, name := range agents {
		agent, err := st.Get(name)
		if err != nil {
			if errors.Is(err, plugin.ErrAgentNotFound) {
				notFound = append(notFound, name)
			} else {
				return nil, err
			}
			continue
		}
		newStore.agents[name] = agent
	}

	if len(notFound) > 0 {
		return nil, plugin.ErrAgentName
	}

	return newStore, nil
}
