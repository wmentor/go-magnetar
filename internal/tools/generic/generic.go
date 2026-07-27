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

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/agent/guard"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/docx"
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
	regexp.MustCompile(`(?i)^\s*su\b`),
	regexp.MustCompile(`(?i)^\s*mkfs\b`),
	regexp.MustCompile(`(?i)^\s*chmod\b`),
	regexp.MustCompile(`(?i)^\s*chown\b`),
	regexp.MustCompile(`(?i)^\s*dscl\b`),
	regexp.MustCompile(`(?i)^\s*fdisk\b`),
	regexp.MustCompile(`(?i)^\s*format\b`),
	regexp.MustCompile(`(?i)^\s*dseditgroup\b`),
	regexp.MustCompile(`(?i)^\s*brew\b`),
	regexp.MustCompile(`(?i)^\s*dpkg\b`),
	regexp.MustCompile(`(?i)^\s*apt\b`),
	regexp.MustCompile(`(?i)^\s*cargo\b`),
	regexp.MustCompile(`(?i)^\s*rpm\b`),
	regexp.MustCompile(`(?i)^\s*npm\b`),
	regexp.MustCompile(`(?i)^\s*ssh\b`),
	regexp.MustCompile(`(?i)^\s*apt\-get\b`),
	regexp.MustCompile(`(?i)^\s*groupadd\b`),
	regexp.MustCompile(`(?i)^\s*usermod\b`),
	regexp.MustCompile(`(?i)^\s*gpasswd\b`),
	regexp.MustCompile(`(?i)^\s*useradd\b`),
	regexp.MustCompile(`(?i)^\s*adduser\b`),
	regexp.MustCompile(`(?i)^\s*userdel\b`),
	regexp.MustCompile(`(?i)^\s*deluser\b`),
	regexp.MustCompile(`(?i)^\s*passwd\b`),
	regexp.MustCompile(`(?i)^\s*systemctl\b`),
	regexp.MustCompile(`(?i)^\s*sysadminctl\b`),
	regexp.MustCompile(`(?i)^\s*dd\s+.*of=/dev/`),
	regexp.MustCompile(`(?i)^\s*git\s+(commit|push|rebase|pull|cherry-pick|amend|reset|stash|clean|reflog|clone)`),
}

var envFilter = regexp.MustCompile(`(?i)(pass|secr|cred|token|key|auth|pwd|cert|sign|salt|bearer|amqp|connect)`)

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

// FileList filter parameter.
type FileListFilter struct {
	Filter string `json:"filter,omitempty"`
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

	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".docx" {
		text, err := docx.ReadFile(filename)
		if err != nil {
			printer.Error("file_read: failed to read docx file", "file", filename, "err", err)
			return "", false
		}
		return text, true
	}

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
	printer.ToolCall(printer.IconSave, "file_write", "file", filename)
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
// filtered by glob pattern.
func (g *GenericTools) FileList(filter *FileListFilter) []string {
	var results []string

	logParams := []any{}
	if filter != nil && filter.Filter != "" {
		logParams = append(logParams, "filter", filter.Filter)
	}

	printer.ToolCall(printer.IconTool, "file_list", logParams...)

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

		if filter != nil && filter.Filter != "" {
			matched, err := filepath.Match(filter.Filter, base)
			if err != nil {
				printer.Error("file_list: invalid filter pattern", "pattern", filter.Filter, "err", err)
				return nil
			}
			if !matched {
				return nil
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

func makeEnvs() []string {
	envs := []string{}
	for _, rec := range os.Environ() {
		pair := strings.SplitN(rec, "=", 2)
		key := pair[0]

		if envFilter.MatchString(key) {
			continue
		}
		envs = append(envs, rec)
	}
	return envs
}

// Exec executes a shell command via "sh -c".
// The command is executed with a clean environment and the current working directory.
// Stdin can be provided via the stdin parameter.
func (g *GenericTools) Exec(command string, stdin string) string {
	printer.ToolCall(printer.IconTool, "exec", "command", command)

	if g.state.ReadOnly {
		printer.ToolCall(printer.IconBlocked, "the exec tool is fobbidden in read-only mode", "command", command)
		return "error: the exec tool is fobbidden in read-only mode"
	}

	if g.isCommandBlocked(command) {
		printer.Print(printer.IconBlocked, "exec: blocked dangerous command", "command", command)
		return "error: dangerous command blocked"
	}

	if !g.cfg.Bool("guard.disable") {
		allowed, reason, err := g.guard.CheckSecurity(command, g.state.ReadOnly)
		if err != nil {
			printer.Print(printer.IconBlocked, "exec: security check failed", "command", command, "err", err)
			return fmt.Sprintf("error: security check failed: %v", err)
		}
		if !allowed {
			if g.cfg.Bool("guard.ask") {
				printer.Print(printer.IconBlocked, "exec: blocked dangerous command", "command", command, "reason", reason)
				if askUserForCommand(command, reason) {
					printer.Print(printer.IconDone, "exec: user confirmed command execution", "command", command)
				} else {
					printer.Print(printer.IconBlocked, "exec: user declined command", "command", command)
					return "error: user declined command execution"
				}
			} else {
				printer.Print(printer.IconBlocked, "exec: blocked dangerous command", "command", command, "reason", reason)
				return "error: security check failed: " + reason
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Dir = g.root.Name()

	cmd.Env = makeEnvs()

	w := newFixedSizeWriter(execOutputMaxSize)
	cmd.Stdout = w
	cmd.Stderr = w

	cmdErr := cmd.Run()
	if cmdErr != nil {
		printer.Error("exec: command failed", "command", command, "error", cmdErr)
		if w.Truncated() {
			return fmt.Sprintf("error: %v\n(output truncated to %d bytes)", cmdErr, execOutputMaxSize)
		}
		return fmt.Sprintf("error: %v\n%s", cmdErr, w.String())
	}

	if w.Truncated() {
		printer.Print(printer.IconAlert, "exec truncated output", "command", command)
		return w.String() + fmt.Sprintf("\n(output truncated to %d bytes)", execOutputMaxSize)
	}

	return w.String()
}

func askUserForCommand(command, reason string) bool {
	fmt.Printf("Security guard blocked this command:\n\n")
	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Reason: %s\n\n", reason)

	var answer string

	for {
		fmt.Printf("Do you want to execute it anyway? (y/N): ")
		answer = ""
		fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "y" || answer == "n" {
			break
		}
	}

	return answer == "y"
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
// pattern (required), always uses -i (case-insensitive), -r (recursive), and -n (line numbers).
func (g *GenericTools) SystemGrep(filename string, pattern string) string {
	printer.ToolCall(printer.IconTool, "system_grep", "pattern", pattern)

	cmd := []string{"grep", "-n", "-i", "-r", "-E", pattern, filename}

	output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code == 1 {
				return "No match found"
			}
		}
		printer.Error("system_grep error", "err", err)
		return fmt.Sprintf("error: %v", err)
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
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
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
			Description: "Execute system grep command with safe parameters: -n (always), -i (case-insensitive), and -r (recursive)",
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
			Filename string   `json:"filename"`
			Limit    *float64 `json:"limit,omitempty"`
			Offset   *float64 `json:"offset,omitempty"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("file_read: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		limit := 0
		offset := 0
		if params.Limit != nil {
			limit = int(*params.Limit)
		}
		if params.Offset != nil {
			offset = int(*params.Offset)
		}
		content, ok := g.FileRead(params.Filename, limit, offset)
		if !ok {
			return "error: failed to read file"
		}
		return content

	case "file_list":
		var params struct {
			Filter string `json:"filter,omitempty"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("file_list: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		results := g.FileList(&FileListFilter{Filter: params.Filter})
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
			Filename string `json:"filename"`
			Pattern  string `json:"pattern"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("system_grep: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		return g.SystemGrep(params.Filename, params.Pattern)

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
			Description: "Execute system grep command with safe parameters: -n (always), -i (case-insensitive), and -r (recursive)",
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
					"filter": map[string]any{
						"type":        "string",
						"description": "Glob pattern to match file names (e.g., '*.go', 'test_*.py')",
					},
				},
			},
		},
	}
}
