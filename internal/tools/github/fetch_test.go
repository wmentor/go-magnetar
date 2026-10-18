package github

import (
	"testing"
)

func TestPullRequestToolDefinition(t *testing.T) {
	t.Parallel()

	tool := StaticDefinitionPullRequest()
	if tool.Type != "function" {
		t.Errorf("expected tool type 'function', got '%s'", tool.Type)
	}

	if tool.Function.Name != "github_pull_request" {
		t.Errorf("expected function name 'github_pull_request', got '%s'", tool.Function.Name)
	}

	if tool.Function.Description == "" {
		t.Error("function description should not be empty")
	}

	// Check required parameters
	params := tool.Function.Parameters.(map[string]any)
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatalf("expected required to be []string")
	}
	if len(required) != 2 {
		t.Error("expected 2 required parameters (repo, pr)")
	}
}

func TestPullRequestFileStruct(t *testing.T) {
	t.Parallel()

	file := PullRequestFile{
		Path:      "README.md",
		Status:    "modified",
		Additions: 10,
		Deletions: 5,
		Changes:   15,
		Diff:      "+10\n-5",
	}

	if file.Path != "README.md" {
		t.Errorf("expected path 'README.md', got '%s'", file.Path)
	}

	if file.Additions != 10 {
		t.Errorf("expected additions 10, got %d", file.Additions)
	}

	if file.Deletions != 5 {
		t.Errorf("expected deletions 5, got %d", file.Deletions)
	}
}

func TestCalculateChanges(t *testing.T) {
	t.Parallel()

	g := &GitHubTools{}

	files := []PullRequestFile{
		{Path: "file1.go", Additions: 10, Deletions: 5},
		{Path: "file2.go", Additions: 20, Deletions: 10},
	}

	additions, deletions, changedFiles := g.calculateChanges(files)

	if additions != 30 {
		t.Errorf("expected total additions 30, got %d", additions)
	}

	if deletions != 15 {
		t.Errorf("expected total deletions 15, got %d", deletions)
	}

	if changedFiles != 2 {
		t.Errorf("expected 2 changed files, got %d", changedFiles)
	}
}
