package generic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

// FileListOptions defines optional filters for listing files.
type FileListOptions struct {
	Extensions []string `json:"extensions,omitempty"`
	Names      []string `json:"names,omitempty"`
}

// GenericTools provides basic file system operations as LLM tools.
type GenericTools struct {
	cfg   *config.Config
	root  *os.Root
	state *plugin.State
}

// New creates a new GenericTools instance.
func New(cfg *config.Config, root *os.Root, state *plugin.State) *GenericTools {
	return &GenericTools{cfg: cfg, root: root, state: state}
}

// Root returns the sandboxed filesystem root used by this instance.
func (g *GenericTools) Root() *os.Root {
	return g.root
}

// FileRead reads a file and returns its content and a success flag.
// If limit > 0, reads at most limit lines starting from offset line.
func (g *GenericTools) FileRead(filename string, limit int, offset int) (string, bool) {
	if limit <= 0 && offset <= 0 {
		data, err := g.root.ReadFile(filename)
		if err != nil {
			slog.Error("file_read: failed to read file", "file", filename, "err", err)
			return "", false
		}
		return string(data), true
	}

	f, err := g.root.Open(filename)
	if err != nil {
		slog.Error("file_read: failed to open file", "file", filename, "err", err)
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var sb strings.Builder
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		if lineCount <= offset {
			continue
		}
		if limit > 0 && lineCount > offset+limit {
			break
		}
		if lineCount > offset+1 {
			sb.WriteByte('\n')
		}
		sb.WriteString(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		slog.Error("file_read: failed to scan file", "file", filename, "err", err)
		return "", false
	}

	return sb.String(), true
}

// FileWrite writes content to a file and returns a success flag.
// Returns false if read-only mode is enabled.
func (g *GenericTools) FileWrite(filename string, content string) bool {
	if g.state.ReadOnly {
		slog.Warn("file_write: blocked by read-only mode", "file", filename)
		return false
	}
	err := g.root.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		slog.Error("file_write: failed to write file", "file", filename, "err", err)
		return false
	}
	slog.Info("file_write: file written successfully", "file", filename)
	return true
}

// FileExists checks if a file exists and returns true if it does.
func (g *GenericTools) FileExists(filename string) bool {
	_, err := g.root.Stat(filename)
	if err != nil {
		slog.Debug("file_exists: file not found", "file", filename, "err", err)
		return false
	}
	slog.Debug("file_exists: file found", "file", filename)
	return true
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
					if strings.EqualFold(ext, e) || strings.EqualFold(ext, "."+e) {
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
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of lines to read (optional, 0 = read all)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Number of lines to skip from the beginning (optional, 0 = start from beginning)",
					},
				},
				"required": []string{"filename"},
			},
		},
	}
}

// DefinitionFileWrite returns the OpenAI tool schema for file_write.
func (g *GenericTools) DefinitionFileWrite() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_write",
			Description: "Write content to a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path where to write the file",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"filename", "content"},
			},
		},
	}
}

// DefinitionFileExists returns the OpenAI tool schema for file_exists.
func (g *GenericTools) DefinitionFileExists() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_exists",
			Description: "Check if a file exists",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to check",
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
			Limit    int    `json:"limit,omitempty"`
			Offset   int    `json:"offset,omitempty"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("file_read: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		content, ok := g.FileRead(params.Filename, params.Limit, params.Offset)
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

	case "file_write":
		var params struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("file_write: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		if ok := g.FileWrite(params.Filename, params.Content); !ok {
			return "error: read only mode"
		}
		return "File written successfully."

	case "file_exists":
		var params struct {
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("file_exists: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		if ok := g.FileExists(params.Filename); !ok {
			return "File does not exist."
		}
		return "File exists."

	default:
		return "error: unknown tool " + name
	}
}

// Static tool definitions — return schemas without requiring a GenericTools instance.

func StaticDefinitionFileRead() openai.Tool {
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
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of lines to read (optional, 0 = read all)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Number of lines to skip from the beginning (optional, 0 = start from beginning)",
					},
				},
				"required": []string{"filename"},
			},
		},
	}
}

func StaticDefinitionFileWrite() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_write",
			Description: "Write content to a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path where to write the file",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"filename", "content"},
			},
		},
	}
}

func StaticDefinitionFileExists() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_exists",
			Description: "Check if a file exists",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to check",
					},
				},
				"required": []string{"filename"},
			},
		},
	}
}

func StaticDefinitionFileList() openai.Tool {
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
