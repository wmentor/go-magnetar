package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

// FileEditParams represents parameters for the file_edit tool.
type FileEditParams struct {
	Filename string `json:"filename" jsonschema_description:"Path to the file to edit"`
	OldText  string `json:"old_text" jsonschema_description:"Text to search for and replace (case-sensitive, must appear exactly once)"`
	NewText  string `json:"new_text" jsonschema_description:"Text to replace with"`
}

// FileEditEdit represents a single edit operation.
type FileEditEdit struct {
	OldText string `json:"old_text" jsonschema_description:"Text to search for and replace (case-sensitive, must match exactly)"`
	NewText string `json:"new_text" jsonschema_description:"Text to replace with"`
}

// FileEditParamsMultiple represents parameters for editing multiple places in a file.
type FileEditParamsMultiple struct {
	Filename string         `json:"filename" jsonschema_description:"Path to the file to edit"`
	Edits    []FileEditEdit `json:"edits" jsonschema_description:"Array of edit operations (case-sensitive search)"`
	Strategy string         `json:"strategy,omitempty" jsonschema_description:"Replacement strategy: 'first' or 'all'"`
}

// Tool provides file editing operations as LLM tools.
type Tool struct {
	cfg     *config.Config
	printer printer.Printer
	root    *os.Root
	state   *plugin.State
}

// NewTool creates a new Tool instance.
func NewTool(cfg *config.Config, p printer.Printer, _ any, root *os.Root, state *plugin.State) *Tool {
	return &Tool{
		cfg:     cfg,
		root:    root,
		state:   state,
		printer: p,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "file_edit"
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return "Edit a plain-text file by replacing text. " +
		"Requires old_text to appear EXACTLY ONCE in the file. CASE-SENSITIVE search. " +
		"NOT supported for binary/structured formats: .docx, .pdf, .odt, .xlsx, .pptx — " +
		"use file_write or a dedicated tool for those."
}

// FileEdit reads a file, replaces occurrences of oldText with newText, and writes the result back.
// Returns a tools.Response with the result string and success flag.
// Note: Uses exact string replacement.
// If oldText appears multiple times, returns an error to force user to use file_edit_multiple.
func (t *Tool) FileEdit(ctx context.Context, params *FileEditParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconTool, "file_edit", "file", params.Filename, "replacements", 1)

	if t.state.ReadOnly {
		t.printer.Error("file_edit: blocked by read-only mode", "file", params.Filename)
		return nil, errors.New("file_edit is disabled in read-only mode")
	}

	// Check security
	if !t.isSafePath(params.Filename) {
		t.printer.Error("file_edit: blocked unsafe path", "file", params.Filename)
		return nil, errors.New("file path is not allowed")
	}

	// Read the file
	content, err := t.readFileContent(params.Filename)
	if err != nil {
		t.printer.Error("file_edit: failed to read file", "file", params.Filename, "err", err)
		return nil, err
	}

	// Check if oldText exists
	if !strings.Contains(content, params.OldText) {
		t.printer.Print(printer.IconTool, "file_edit: old_text not found in file", "file", params.Filename)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "warning: old_text not found in file, no changes made",
		}, nil
	}

	// Count occurrences - requires exact match and unique occurrence
	count := strings.Count(content, params.OldText)
	if count > 1 {
		t.printer.Error("file_edit: old_text appears multiple times", "file", params.Filename, "count", count)
		return nil, fmt.Errorf("file_edit: old_text appears %d times in the file", count)
	}

	t.outputDiff(params.OldText, params.NewText)

	// Perform replacement (single occurrence)
	newContent := strings.Replace(content, params.OldText, params.NewText, 1)

	// Write the file back
	err = t.writeFileContent(params.Filename, newContent)
	if err != nil {
		t.printer.Error("file_edit: failed to write file", "file", params.Filename, "err", err)
		return nil, fmt.Errorf("file_edit: failed to write file: %w", err)
	}

	t.printer.ToolCall(printer.IconDone, "file_edit: file edited successfully", "file", params.Filename)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: "File edited successfully. Changes have been saved.",
	}, nil
}

// FileEditMultiple reads a file, applies multiple edits, and writes the result back.
// Returns a tools.Response with the result string and success flag.
// Note: Each edit in the array will replace ALL occurrences of its old_text by default.
func (t *Tool) FileEditMultiple(ctx context.Context, params *FileEditParamsMultiple) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconTool, "file_edit_multiple", "file", params.Filename, "edits_count", len(params.Edits))

	if t.state.ReadOnly {
		t.printer.Error("file_edit_multiple: blocked by read-only mode", "file", params.Filename)
		return nil, errors.New("file_edit_multiple is disabled in read-only mode")
	}

	// Check security
	if !t.isSafePath(params.Filename) {
		t.printer.Error("file_edit_multiple: blocked unsafe path", "file", params.Filename)
		return nil, errors.New("file path is not allowed")
	}

	// Read the file
	content, err := t.readFileContent(params.Filename)
	if err != nil {
		t.printer.Error("file_edit_multiple: failed to read file", "file", params.Filename, "err", err)
		return nil, err
	}

	newContent := content
	editsApplied := 0

	for _, edit := range params.Edits {
		if params.Strategy == "first" {
			// Replace only the first occurrence
			if strings.Contains(newContent, edit.OldText) {
				newContent = strings.Replace(newContent, edit.OldText, edit.NewText, 1)
				editsApplied++
				t.outputDiff(edit.OldText, edit.NewText)
			}
		} else {
			// Replace all occurrences (default behavior)
			if strings.Contains(newContent, edit.OldText) {
				newContent = strings.ReplaceAll(newContent, edit.OldText, edit.NewText)
				editsApplied++
				t.outputDiff(edit.OldText, edit.NewText)
			}
		}
	}

	// Check if any edits were actually applied
	if editsApplied == 0 {
		t.printer.Print(printer.IconTool, "file_edit_multiple: no edits applied", "file", params.Filename)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "warning: no changes were applied (old_text not found in file)",
		}, nil
	}

	// Write the file back
	err = t.writeFileContent(params.Filename, newContent)
	if err != nil {
		t.printer.Error("file_edit_multiple: failed to write file", "file", params.Filename, "err", err)
		return nil, fmt.Errorf("file_edit_multiple: failed to write file: %w", err)
	}

	t.printer.ToolCall(printer.IconDone, "file_edit_multiple: file edited successfully", "file", params.Filename, "edits_applied", editsApplied)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: "File edited successfully. " + strconv.Itoa(editsApplied) + " changes have been saved.",
	}, nil
}

// isSafePath checks if the file path is safe to edit.
func (t *Tool) isSafePath(filename string) bool {
	// Prevent editing files outside the root directory
	// First, try to get the absolute path from the root
	var absPath string
	var err error

	if t.root != nil {
		// Resolve relative to root
		rootName := t.root.Name()
		if filepath.IsAbs(filename) {
			absPath, err = filepath.Abs(filename)
		} else {
			absPath = filepath.Join(rootName, filename)
		}
	} else {
		// Fallback to current directory
		absPath, err = filepath.Abs(filename)
	}

	if err != nil {
		return false
	}

	if t.root == nil {
		return false
	}

	rootPath := t.root.Name()
	relPath, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		return false
	}

	// Check if the path is within root
	if strings.HasPrefix(relPath, "../") {
		return false
	}

	// Block dangerous file patterns
	dangerousPatterns := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/hosts",
		"/etc/ssh/",
		"/.git/",
		".ssh/",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(absPath, pattern) {
			return false
		}
	}

	return true
}

// readFileContent reads file content as raw bytes.
// Returns an error for formats handled by a codec (.docx, .pdf, .odt, .xlsx, .pptx, .csv, .tsv, .html, .htm)
// because those are binary or structured formats that cannot be round-tripped through text replacement.
// Use file_write or a dedicated tool for such formats.
func (t *Tool) readFileContent(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, err := codec.GetCodec(ext); err == nil {
		return "", fmt.Errorf(
			"file_edit cannot edit %s files: use file_write or a dedicated tool for this format", ext,
		)
	}

	data, err := t.root.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeFileContent writes content to file.
func (t *Tool) writeFileContent(filename string, content string) error {
	// Create directories if they don't exist
	dir := filepath.Dir(filename)
	if err := t.root.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return t.root.WriteFile(filename, []byte(content), 0o644)
}

// DefinitionFileEdit returns the OpenAI tool schema for file_edit.
func (t *Tool) DefinitionFileEdit() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "file_edit",
			Description: "Edit a plain-text file by replacing text. " +
				"Requires old_text to appear EXACTLY ONCE in the file. CASE-SENSITIVE search. " +
				"NOT supported for binary/structured formats: .docx, .pdf, .odt, .xlsx, .pptx, .csv, .tsv, .html, .htm — " +
				"use file_write or a dedicated tool for those.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"old_text": map[string]any{
						"type":        "string",
						"description": "Text to search for and replace (case-sensitive, must appear exactly once)",
					},
					"new_text": map[string]any{
						"type":        "string",
						"description": "Text to replace with",
					},
				},
				"required": []string{"filename", "old_text", "new_text"},
			},
		},
	}
}

// DefinitionFileEditMultiple returns the OpenAI tool schema for file_edit_multiple.
func (t *Tool) DefinitionFileEditMultiple() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "file_edit_multiple",
			Description: "Edit a plain-text file by applying multiple text replacements (case-sensitive). " +
				"Can replace first occurrence or all occurrences. " +
				"NOT supported for binary/structured formats: .docx, .pdf, .odt, .xlsx, .pptx, .csv, .tsv, .html, .htm.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"edits": map[string]any{
						"type":        "array",
						"description": "Array of edit operations (case-sensitive search)",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"old_text": map[string]any{
									"type":        "string",
									"description": "Text to search for and replace (case-sensitive, must match exactly)",
								},
								"new_text": map[string]any{
									"type":        "string",
									"description": "Text to replace with",
								},
							},
							"required": []string{"old_text", "new_text"},
						},
					},
					"strategy": map[string]any{
						"type":        "string",
						"description": "Replacement strategy: 'first' to replace only the first occurrence, 'all' to replace all occurrences (default: 'all')",
						"enum":        []string{"first", "all"},
					},
				},
				"required": []string{"filename", "edits"},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (t *Tool) Dispatch(name string, args string) string {
	switch name {
	case "file_edit":
		var params FileEditParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			t.printer.Error("file_edit: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		result, err := t.FileEdit(context.Background(), &params)
		if err != nil {
			return err.Error()
		}
		if resp, ok := result.(*tools.Response); ok {
			return resp.Result
		}
		return fmt.Sprintf("%v", result)

	case "file_edit_multiple":
		var params FileEditParamsMultiple
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			t.printer.Error("file_edit_multiple: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		// Default strategy to "all" if not specified
		if params.Strategy == "" {
			params.Strategy = "all"
		}
		result, err := t.FileEditMultiple(context.Background(), &params)
		if err != nil {
			return err.Error()
		}
		if resp, ok := result.(*tools.Response); ok {
			return resp.Result
		}
		return fmt.Sprintf("%v", result)

	default:
		return "error: unknown tool " + name
	}
}

// StaticDefinitionFileEdit returns the OpenAI tool schema for file_edit without requiring an instance.
func StaticDefinitionFileEdit() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "file_edit",
			Description: "Edit a plain-text file by replacing text. " +
				"Requires old_text to appear EXACTLY ONCE in the file. CASE-SENSITIVE search. " +
				"NOT supported for binary/structured formats: .docx, .pdf, .odt, .xlsx, .pptx, .csv, .tsv, .html, .htm — " +
				"use file_write or a dedicated tool for those.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"old_text": map[string]any{
						"type":        "string",
						"description": "Text to search for and replace (case-sensitive, must appear exactly once)",
					},
					"new_text": map[string]any{
						"type":        "string",
						"description": "Text to replace with",
					},
				},
				"required": []string{"filename", "old_text", "new_text"},
			},
		},
	}
}

// StaticDefinitionFileEditMultiple returns the OpenAI tool schema for file_edit_multiple without requiring an instance.
func StaticDefinitionFileEditMultiple() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "file_edit_multiple",
			Description: "Edit a plain-text file by applying multiple text replacements (case-sensitive). " +
				"Can replace first occurrence or all occurrences. " +
				"NOT supported for binary/structured formats: .docx, .pdf, .odt, .xlsx, .pptx, .csv, .tsv, .html, .htm.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"edits": map[string]any{
						"type":        "array",
						"description": "Array of edit operations (case-sensitive search)",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"old_text": map[string]any{
									"type":        "string",
									"description": "Text to search for and replace (case-sensitive, must match exactly)",
								},
								"new_text": map[string]any{
									"type":        "string",
									"description": "Text to replace with",
								},
							},
							"required": []string{"old_text", "new_text"},
						},
					},
					"strategy": map[string]any{
						"type":        "string",
						"description": "Replacement strategy: 'first' to replace only the first occurrence, 'all' to replace all occurrences (default: 'all')",
						"enum":        []string{"first", "all"},
					},
				},
				"required": []string{"filename", "edits"},
			},
		},
	}
}

func (t *Tool) outputDiff(oldText string, newText string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, false)
	t.printer.PrintRaw(dmp.DiffPrettyText(diffs))
}
