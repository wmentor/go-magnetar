package generic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/agent/guard"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const execTimeout = 1 * time.Minute
const execOutputMaxSize = 64 * 1024

var blockList = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)*(/\s*$|/\s+)`),
	regexp.MustCompile(`(?i)\|\s*bash\b`),
	regexp.MustCompile(`(?i)\|\s*sh\b`),
	regexp.MustCompile(`(?i)\|\s*zsh\b`),
	regexp.MustCompile(`(?i)^\s*sudo\b`),
	regexp.MustCompile(`(?i)^\s*mkfs\b`),
	regexp.MustCompile(`(?i)^\s*dd\s+.*of=/dev/`),
	regexp.MustCompile(`(?i)^\s*git\s+(commit|push|rebase|pull|cherry-pick|amend|reset|stash|clean|reflog)`),
}

type fixedSizeWriter struct {
	data    []byte
	written int
	limit   int
	full    bool
}

func newFixedSizeWriter(limit int) *fixedSizeWriter {
	return &fixedSizeWriter{limit: limit, data: make([]byte, limit)}
}

func (w *fixedSizeWriter) Write(p []byte) (int, error) {
	if w.full {
		return len(p), nil
	}

	ret := len(p)

	remain := w.limit - w.written
	if len(p) > remain {
		p = p[:remain]
		w.full = true
	}

	n := copy(w.data[w.written:], p)
	w.written += n
	return ret, nil
}

func (w *fixedSizeWriter) String() string {
	return string(w.data[:w.written])
}

func (w *fixedSizeWriter) Truncated() bool {
	return w.full
}

func (g *GenericTools) isCommandBlocked(command string) bool {
	for _, re := range blockList {
		if re.MatchString(command) {
			return true
		}
	}
	return false
}

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
	guard *guard.Guard
}

// New creates a new GenericTools instance.
func New(cfg *config.Config, root *os.Root, state *plugin.State) *GenericTools {
	return &GenericTools{
		cfg:   cfg,
		root:  root,
		state: state,
		guard: guard.New(cfg),
	}
}

// Root returns the sandboxed filesystem root used by this instance.
func (g *GenericTools) Root() *os.Root {
	return g.root
}

// FileRead reads a file and returns its content and a success flag.
// If limit > 0, reads at most limit lines starting from offset line.
func (g *GenericTools) FileRead(filename string, limit int, offset int) (string, bool) {
	printer.ToolCall(printer.IconTool, "file_read", "filename", filename)

	if limit <= 0 && offset <= 0 {
		data, err := g.root.ReadFile(filename)
		if err != nil {
			printer.Error("file_read: failed to read file", "file", filename, "err", err)
			return "", false
		}
		return string(data), true
	}

	f, err := g.root.Open(filename)
	if err != nil {
		printer.Error("file_read: failed to open file", "file", filename, "err", err)
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
		printer.Error("file_read: failed to scan file", "file", filename, "err", err)
		return "", false
	}

	return sb.String(), true
}

// FileWrite writes content to a file and returns a success flag.
// Returns false if read-only mode is enabled.
func (g *GenericTools) FileWrite(filename string, content string) bool {
	printer.ToolCall(printer.IconSave, "file_write: file written successfully", "file", filename)
	if g.state.ReadOnly {
		printer.Error("file_write: blocked by read-only mode", "file", filename)
		return false
	}
	err := g.root.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		printer.Error("file_write: failed to write file", "file", filename, "err", err)
		return false
	}
	return true
}

// FileExists checks if a file exists and returns true if it does.
func (g *GenericTools) FileExists(filename string) bool {
	_, err := g.root.Stat(filename)
	if err != nil {
		printer.Debug("file_exists: file not found", "file", filename, "err", err)
		return false
	}
	printer.Debug("file_exists: file found", "file", filename)
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
		printer.Error("file_list: failed to walk directory", "path", g.root.Name(), "err", err)
		return []string{fmt.Sprintf("error: %v", err)}
	}

	return results
}

// Exec executes a shell command via "sh -c".
// The command is executed with a clean environment and the current working directory.
// Stdin can be provided via the stdin parameter.
func (g *GenericTools) Exec(command string, stdin string) string {
	printer.ToolCall(printer.IconTool, "exec", "command", command)

	if g.isCommandBlocked(command) {
		printer.Print(printer.IconBlocked, "exec: blocked dangerous command", "command", command)
		return "error: dangerous command blocked"
	}

	allowed, reason, err := g.guard.CheckSecurity(command, g.state.ReadOnly)
	if err != nil {
		printer.Print(printer.IconBlocked, "exec: security check failed", "command", command, "err", err)
		return fmt.Sprintf("error: security check failed: %v", err)
	}
	if !allowed {
		printer.Print(printer.IconBlocked, "exec: security blocked", "command", command, "reason", reason)
		return "error: security check failed: " + reason
	}

	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Dir = g.root.Name()

	env := []string{}
	cmd.Env = env

	w := newFixedSizeWriter(execOutputMaxSize)
	cmd.Stdout = w
	cmd.Stderr = w

	err = cmd.Run()
	if err != nil {
		printer.Error("exec: command failed", "command", command, "error", err)
		if w.Truncated() {
			return fmt.Sprintf("error: %v\n(output truncated to %d bytes)", err, execOutputMaxSize)
		}
		return fmt.Sprintf("error: %v\n%s", err, w.String())
	}

	if w.Truncated() {
		printer.Print(printer.IconAlert, "exec truncated output", "command", command)
		return w.String() + fmt.Sprintf("\n(output truncated to %d bytes)", execOutputMaxSize)
	}

	return w.String()
}

// SystemDate executes the date command and returns the output.
func (g *GenericTools) SystemDate() string {
	cmd := exec.Command("date")

	output, err := cmd.Output()
	if err != nil {
		printer.Error("system_date: command failed", "err", err, "output", string(output))
		return fmt.Sprintf("error: %v\n%s", err, string(output))
	}

	return string(strings.TrimSpace(string(output)))
}

// SystemGrep executes the system grep command with a limited set of safe arguments:
// pattern (required), -i (case-insensitive), -r (recursive), and always adds -n (line numbers).
func (g *GenericTools) SystemGrep(filename string, pattern string, caseInsensitive bool, recursive bool) string {
	printer.ToolCall(printer.IconTool, "system_grep", "pattern", pattern)

	cmd := []string{"grep", "-n"}

	if caseInsensitive {
		cmd = append(cmd, "-i")
	}

	if recursive {
		cmd = append(cmd, "-r")
	}

	cmd = append(cmd, "-E", pattern, filename)

	output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		printer.Error("system_grep: command failed", "cmd", cmd, "err", err, "output", string(output))
		return fmt.Sprintf("error: %v\n%s", err, output)
	}

	return string(output)
}

// SearchReplace applies search-and-replace operations to a file.
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

// DefinitionFileGrep returns the OpenAI tool schema for file_grep.
func (g *GenericTools) DefinitionFileGrep() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_grep",
			Description: "Search for a regex pattern in a file and return matching lines with line numbers",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to search",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "Regular expression pattern to search for",
					},
				},
				"required": []string{"filename", "pattern"},
			},
		},
	}
}

// DefinitionExec returns the OpenAI tool schema for exec.
func (g *GenericTools) DefinitionExec() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "exec",
			Description: "Execute a shell command via \"sh -c\". The command runs with a clean environment and the current working directory. Stdin can be provided for input.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute",
					},
					"stdin": map[string]any{
						"type":        "string",
						"description": "Standard input to pass to the command (optional)",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// DefinitionSystemDate returns the OpenAI tool schema for system_date.
func (g *GenericTools) DefinitionSystemDate() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "system_date",
			Description: "Execute the date command to get the current system time",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}
}

// DefinitionSystemGrep returns the OpenAI tool schema for system_grep.
func (g *GenericTools) DefinitionSystemGrep() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "system_grep",
			Description: "Execute system grep command with safe parameters: -n (always), optional -i (case-insensitive) and -r (recursive)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file or directory to search",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "Regular expression pattern to search for",
					},
					"case_insensitive": map[string]any{
						"type":        "boolean",
						"description": "If true, perform case-insensitive matching (default: false)",
					},
					"recursive": map[string]any{
						"type":        "boolean",
						"description": "If true, search recursively in directories (default: false)",
					},
				},
				"required": []string{"filename", "pattern"},
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
			printer.Error("file_read: failed to parse args", "args", args, "err", err)
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
			printer.Error("file_list: failed to parse args", "args", args, "err", err)
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
			printer.Error("file_write: failed to parse args", "args", args, "err", err)
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
			printer.Error("file_exists: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		if ok := g.FileExists(params.Filename); !ok {
			return "File does not exist."
		}
		return "File exists."

	case "exec":
		var params struct {
			Command string `json:"command"`
			Stdin   string `json:"stdin,omitempty"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("exec: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		return g.Exec(params.Command, params.Stdin)

	case "system_date":
		return g.SystemDate()

	case "system_grep":
		var params struct {
			Filename        string `json:"filename"`
			Pattern         string `json:"pattern"`
			CaseInsensitive bool   `json:"case_insensitive,omitempty"`
			Recursive       bool   `json:"recursive,omitempty"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("system_grep: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		return g.SystemGrep(params.Filename, params.Pattern, params.CaseInsensitive, params.Recursive)

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

// StaticDefinitionExec returns the OpenAI tool schema for exec without instance.
func StaticDefinitionExec() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "exec",
			Description: "Execute a shell command via \"sh -c\". The command runs with a clean environment and the current working directory. Stdin can be provided for input.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute",
					},
					"stdin": map[string]any{
						"type":        "string",
						"description": "Standard input to pass to the command (optional)",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// StaticDefinitionSystemDate returns the OpenAI tool schema for system_date without instance.
func StaticDefinitionSystemDate() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "system_date",
			Description: "Execute the date command to get the current system time",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}
}

// StaticDefinitionSystemGrep returns the OpenAI tool schema for system_grep without instance.
func StaticDefinitionSystemGrep() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "system_grep",
			Description: "Execute system grep command with safe parameters: -n (always), optional -i (case-insensitive) and -r (recursive)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file or directory to search",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "Regular expression pattern to search for",
					},
					"case_insensitive": map[string]any{
						"type":        "boolean",
						"description": "If true, perform case-insensitive matching (default: false)",
					},
					"recursive": map[string]any{
						"type":        "boolean",
						"description": "If true, search recursively in directories (default: false)",
					},
				},
				"required": []string{"filename", "pattern"},
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
