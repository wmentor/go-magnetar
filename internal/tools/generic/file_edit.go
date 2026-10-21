package generic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/wmentor/go-magnetar/internal/agent/guard"
	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

// FileEditParams represents parameters for the file_edit tool.
type FileEditParams struct {
	Filename string `json:"filename"`
	OldText  string `json:"old_text"`
	NewText  string `json:"new_text"`
}

// FileEditEdit represents a single edit operation.
type FileEditEdit struct {
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

// FileEditParamsMultiple represents parameters for editing multiple places in a file.
type FileEditParamsMultiple struct {
	Filename string         `json:"filename"`
	Edits    []FileEditEdit `json:"edits"`
	Strategy string         `json:"strategy,omitempty"` // "first" or "all"
}

// FileEditTools provides file editing operations as LLM tools.
type FileEditTools struct {
	cfg   *config.Config
	root  *os.Root
	state *plugin.State
	guard *guard.Guard
}

// NewFileEditTools creates a new FileEditTools instance.
func NewFileEditTools(cfg *config.Config, root *os.Root, state *plugin.State) *FileEditTools {
	return &FileEditTools{
		cfg:   cfg,
		root:  root,
		state: state,
		guard: guard.New(cfg),
	}
}

// FileEdit reads a file, replaces occurrences of oldText with newText, and writes the result back.
// Returns true if successful, false otherwise.
// Note: Uses exact string replacement.
// If oldText appears multiple times, returns an error to force user to use file_edit_multiple.
func (f *FileEditTools) FileEdit(filename string, oldText string, newText string) (string, bool) {
	printer.ToolCall(printer.IconTool, "file_edit", "file", filename, "replacements", 1)

	if f.state.ReadOnly {
		printer.Error("file_edit: blocked by read-only mode", "file", filename)
		return "error: file_edit is disabled in read-only mode", false
	}

	// Check security
	if !f.isSafePath(filename) {
		printer.Error("file_edit: blocked unsafe path", "file", filename)
		return "error: file path is not allowed", false
	}

	// Read the file
	content, err := f.readFileContent(filename)
	if err != nil {
		printer.Error("file_edit: failed to read file", "file", filename, "err", err)
		return "error: failed to read file: " + err.Error(), false
	}

	// Check if oldText exists
	if !strings.Contains(content, oldText) {
		printer.Print(printer.IconTool, "file_edit: old_text not found in file", "file", filename)
		return "warning: old_text not found in file, no changes made", true
	}

	// Count occurrences - requires exact match and unique occurrence
	count := strings.Count(content, oldText)
	if count > 1 {
		printer.Error("file_edit: old_text appears multiple times", "file", filename, "count", count)
		return fmt.Sprintf("error: old_text appears %d times in the file.", count), false
	}

	outputDiff(printer.GetDefault(), oldText, newText)

	// Perform replacement (single occurrence)
	newContent := strings.Replace(content, oldText, newText, 1)

	// Write the file back
	err = f.writeFileContent(filename, newContent)
	if err != nil {
		printer.Error("file_edit: failed to write file", "file", filename, "err", err)
		return "error: failed to write file: " + err.Error(), false
	}

	printer.ToolCall(printer.IconDone, "file_edit: file edited successfully", "file", filename)
	return "File edited successfully. Changes have been saved.", true
}

// FileEditMultiple reads a file, applies multiple edits, and writes the result back.
// Returns true if successful, false otherwise.
// Note: Each edit in the array will replace ALL occurrences of its old_text by default.
func (f *FileEditTools) FileEditMultiple(filename string, edits []FileEditEdit, strategy string) (string, bool) {
	printer.ToolCall(printer.IconTool, "file_edit_multiple", "file", filename, "edits_count", len(edits))

	if f.state.ReadOnly {
		printer.Error("file_edit_multiple: blocked by read-only mode", "file", filename)
		return "error: file_edit_multiple is disabled in read-only mode", false
	}

	// Check security
	if !f.isSafePath(filename) {
		printer.Error("file_edit_multiple: blocked unsafe path", "file", filename)
		return "error: file path is not allowed", false
	}

	// Read the file
	content, err := f.readFileContent(filename)
	if err != nil {
		printer.Error("file_edit_multiple: failed to read file", "file", filename, "err", err)
		return "error: failed to read file: " + err.Error(), false
	}

	newContent := content
	editsApplied := 0

	for _, edit := range edits {
		if strategy == "first" {
			// Replace only the first occurrence
			if strings.Contains(newContent, edit.OldText) {
				newContent = strings.Replace(newContent, edit.OldText, edit.NewText, 1)
				editsApplied++
				outputDiff(printer.GetDefault(), edit.OldText, edit.NewText)
			}
		} else {
			// Replace all occurrences (default behavior)
			if strings.Contains(newContent, edit.OldText) {
				newContent = strings.ReplaceAll(newContent, edit.OldText, edit.NewText)
				editsApplied++
				outputDiff(printer.GetDefault(), edit.OldText, edit.NewText)
			}
		}
	}

	// Check if any edits were actually applied
	if editsApplied == 0 {
		printer.Print(printer.IconTool, "file_edit_multiple: no edits applied", "file", filename)
		return "warning: no changes were applied (old_text not found in file)", true
	}

	// Write the file back
	err = f.writeFileContent(filename, newContent)
	if err != nil {
		printer.Error("file_edit_multiple: failed to write file", "file", filename, "err", err)
		return "error: failed to write file: " + err.Error(), false
	}

	printer.ToolCall(printer.IconDone, "file_edit_multiple: file edited successfully", "file", filename, "edits_applied", editsApplied)
	return "File edited successfully. " + strconv.Itoa(editsApplied) + " changes have been saved.", true
}

// isSafePath checks if the file path is safe to edit.
func (f *FileEditTools) isSafePath(filename string) bool {
	// Prevent editing files outside the root directory
	// First, try to get the absolute path from the root
	var absPath string
	var err error

	if f.root != nil {
		// Resolve relative to root
		rootName := f.root.Name()
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

	if f.root == nil {
		return false
	}

	rootPath := f.root.Name()
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
func (f *FileEditTools) readFileContent(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, err := codec.GetCodec(ext); err == nil {
		return "", fmt.Errorf(
			"file_edit cannot edit %s files: use file_write or a dedicated tool for this format", ext,
		)
	}

	data, err := f.root.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeFileContent writes content to file.
func (f *FileEditTools) writeFileContent(filename string, content string) error {
	// Create directories if they don't exist
	dir := filepath.Dir(filename)
	if err := f.root.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return f.root.WriteFile(filename, []byte(content), 0o644)
}

// DefinitionFileEdit returns the OpenAI tool schema for file_edit.
func (f *FileEditTools) DefinitionFileEdit() openai.Tool {
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
func (f *FileEditTools) DefinitionFileEditMultiple() openai.Tool {
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
func (f *FileEditTools) Dispatch(name string, args string) string {
	switch name {
	case "file_edit":
		var params FileEditParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("file_edit: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		result, ok := f.FileEdit(params.Filename, params.OldText, params.NewText)
		if !ok {
			return result // error message
		}
		return result

	case "file_edit_multiple":
		var params FileEditParamsMultiple
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.Error("file_edit_multiple: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		// Default strategy to "all" if not specified
		if params.Strategy == "" {
			params.Strategy = "all"
		}
		result, ok := f.FileEditMultiple(params.Filename, params.Edits, params.Strategy)
		if !ok {
			return result // error message
		}
		return result

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

func outputDiff(p printer.Printer, oldText string, newText string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, false)
	p.PrintRaw(dmp.DiffPrettyText(diffs))
}
