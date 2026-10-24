package writeplugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWriteUsageWhenNoFilename(t *testing.T) {
	t.Helper()
	out := RunWrite("", nil).Stdout
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage, got %q", out)
	}
}

func TestRunWriteRequiresHistory(t *testing.T) {
	t.Helper()
	out := RunWrite("out.md", []Message{{Role: "user", Content: "hi"}}).Stdout
	if !strings.Contains(out, "no conversation history") {
		t.Fatalf("expected history message, got %q", out)
	}
}

func TestRunWriteRejectsEmptyAssistantMessage(t *testing.T) {
	t.Helper()
	out := RunWrite("out.md", []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: ""},
	}).Stdout
	if !strings.Contains(out, "empty") {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestRunWriteSavesLastAssistantAnswer(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "answer.md")
	out := RunWrite(path, []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "# Title\n\nBody"},
	}).Stdout
	if !strings.Contains(out, "Answer saved") {
		t.Fatalf("expected success markdown, got %q", out)
	}
	if !strings.Contains(out, path) {
		t.Fatalf("expected file path in diagnostics, got %q", out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Title\n\nBody" {
		t.Fatalf("content = %q", data)
	}
}

func TestRunWriteFallsBackToPreviousAssistantTurn(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback.md")
	out := RunWrite(path, []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "ok"},
	}).Stdout
	if !strings.Contains(out, "Answer saved") {
		t.Fatalf("expected success, got %q", out)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "first answer" {
		t.Fatalf("content = %q", data)
	}
}

func TestRunWriteReportsInvalidPath(t *testing.T) {
	t.Helper()
	out := RunWrite(filepath.Join(t.TempDir(), "missing", "out.md"), []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "body"},
	}).Stdout
	if !strings.Contains(out, "Error saving file") {
		t.Fatalf("expected error markdown, got %q", out)
	}
}

func TestRunWriteHandlesSpecialCharacters(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "special.md")
	content := "```\ncode\n```\n**bold** | pipe\n"
	out := RunWrite(path, []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: content},
	}).Stdout
	if !strings.Contains(out, "Answer saved") {
		t.Fatalf("expected success, got %q", out)
	}
	data, _ := os.ReadFile(path)
	if string(data) != content {
		t.Fatalf("content = %q", data)
	}
}
