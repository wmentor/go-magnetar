package generic

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
)

// FileListOptions defines optional filters for listing files.
type FileListOptions struct {
	Extensions []string `json:"extensions,omitempty"`
	Names      []string `json:"names,omitempty"`
}

// GenericTools provides basic file system operations as LLM tools.
type GenericTools struct {
	cfg  *config.Config
	root *os.Root
}

// New creates a new GenericTools instance.
func New(cfg *config.Config, root *os.Root) *GenericTools {
	return &GenericTools{cfg: cfg, root: root}
}

// FileRead reads a file and returns its content and a success flag.
func (g *GenericTools) FileRead(filename string) (string, bool) {
	data, err := g.root.ReadFile(filename)
	if err != nil {
		slog.Error("file_read: failed to read file", "file", filename, "err", err)
		return "", false
	}
	return string(data), true
}

// FileList returns a list of files in the current directory recursively,
// filtered by name and/or extension.
func (g *GenericTools) FileList(opts *FileListOptions) []string {
	var results []string

	err := filepath.WalkDir(g.root.Name(), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(g.root.Name(), p)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		base := filepath.Base(p)

		if d.IsDir() {
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if opts != nil {
			if len(opts.Extensions) > 0 {
				ext := filepath.Ext(p)
				matched := false
				for _, e := range opts.Extensions {
					if strings.EqualFold(ext, e) {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}

			if len(opts.Names) > 0 {
				matched := false
				for _, n := range opts.Names {
					if strings.EqualFold(base, n) {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}
		}

		results = append(results, relPath)
		return nil
	})

	if err != nil {
		slog.Error("file_list: failed to walk directory", "path", g.root.Name(), "err", err)
		return []string{fmt.Sprintf("error: %v", err)}
	}

	return results
}

// DefinitionFileRead returns the OpenAI tool schema for file_read.
func (g *GenericTools) DefinitionFileRead() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_read",
			Description: "Read the contents of a file by its path",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"filename"},
			},
		},
	}
}

// DefinitionFileList returns the OpenAI tool schema for file_list.
func (g *GenericTools) DefinitionFileList() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_list",
			Description: "List all files in the current directory tree",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"options": map[string]any{
						"type":        "object",
						"description": "Optional filtering criteria",
						"properties": map[string]any{
							"extensions": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "string"},
								"description": "File extensions to include (e.g., ['.md', '.txt'])",
							},
							"names": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "string"},
								"description": "File names to include (exact matches)",
							},
						},
					},
				},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (g *GenericTools) Dispatch(name string, args string) string {
	switch name {
	case "file_read":
		var params struct {
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("file_read: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		content, ok := g.FileRead(params.Filename)
		if !ok {
			return "error: failed to read file"
		}
		return content

	case "file_list":
		var params struct {
			Options *FileListOptions `json:"options,omitempty"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("file_list: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		results := g.FileList(params.Options)
		return strings.Join(results, "\n")

	default:
		return "error: unknown tool " + name
	}
}
