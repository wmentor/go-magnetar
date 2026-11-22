package initplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.init", &Plugin{})
}

// Plugin registers the /init chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "init",
		Help: "Initialize project documentation (AGENTS.md) by analyzing codebase structure.",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			dir := "."
			if params != nil && strings.TrimSpace(params.Args) != "" {
				dir = strings.TrimSpace(params.Args)
			}

			// 1. Check if AGENTS.md or CLAUDE.md already exists
			for _, file := range []string{"AGENTS.md", "CLAUDE.md"} {
				target := filepath.Join(dir, file)
				if _, err := os.Stat(target); err == nil {
					return &plugin.ChatCommandResponse{
						Stdout: fmt.Sprintf("Project is already initialized: %s already exists.\n", file),
					}, nil
				}
			}

			// 2. Scan directory for structure and key files
			topLevel, keyFiles, err := scanProject(dir)
			if err != nil {
				return &plugin.ChatCommandResponse{
					Stdout: fmt.Sprintf("Failed to scan project directory: %v\n", err),
				}, nil
			}

			// 3. Build model prompt in Content
			var keyFilesStr string
			if len(keyFiles) > 0 {
				keyFilesStr = strings.Join(keyFiles, ", ")
			} else {
				keyFilesStr = "none detected"
			}

			var topLevelStr string
			if len(topLevel) > 0 {
				topLevelStr = strings.Join(topLevel, ", ")
			} else {
				topLevelStr = "empty directory"
			}

			content := fmt.Sprintf(`Please initialize this project by analyzing the codebase and creating a comprehensive AGENTS.md file in the project root.

Detected project structure:
- Key configuration and build files: %s
- Top-level files and directories: %s

Please:
1. Examine the codebase structure, dependencies, build/test workflows, and key conventions.
2. Generate and write an AGENTS.md file in the project root with the following sections:
   - Project overview and purpose
   - Architecture and component structure
   - Build, run, and test commands
   - Development conventions and guidelines
Use the file_write tool to save the resulting AGENTS.md file.`, keyFilesStr, topLevelStr)

			return &plugin.ChatCommandResponse{
				Stdout:  "Initializing project: analyzing structure and generating AGENTS.md...\n",
				Content: content,
			}, nil
		},
	})
	return nil
}

var keyFileNames = map[string]bool{
	"go.mod":              true,
	"go.sum":              true,
	"package.json":        true,
	"tsconfig.json":       true,
	"pnpm-lock.yaml":      true,
	"yarn.lock":           true,
	"Cargo.toml":          true,
	"Cargo.lock":          true,
	"pyproject.toml":      true,
	"requirements.txt":    true,
	"Pipfile":             true,
	"setup.py":            true,
	"pom.xml":             true,
	"build.gradle":        true,
	"build.gradle.kts":    true,
	"CMakeLists.txt":      true,
	"Makefile":            true,
	"Dockerfile":          true,
	"docker-compose.yml":  true,
	"docker-compose.yaml": true,
	"README.md":           true,
	"LICENSE":             true,
}

var ignoredDirs = map[string]bool{
	".git":         true,
	".idea":        true,
	".vscode":      true,
	".venv":        true,
	"venv":         true,
	"node_modules": true,
	"vendor":       true,
	"target":       true,
	"dist":         true,
	"build":        true,
}

func scanProject(dir string) ([]string, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	var topLevel []string
	var keyFiles []string

	for _, entry := range entries {
		name := entry.Name()
		if ignoredDirs[name] {
			continue
		}

		if entry.IsDir() {
			topLevel = append(topLevel, name+"/")
		} else {
			topLevel = append(topLevel, name)
			if keyFileNames[name] {
				keyFiles = append(keyFiles, name)
			}
		}
	}

	sort.Strings(topLevel)
	sort.Strings(keyFiles)

	return topLevel, keyFiles, nil
}
