package writeplugin

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

// fakeAgentHandle is a minimal plugin.AgentHandle for testing /write.
type fakeAgentHandle struct {
	msgs []openai.ChatCompletionMessage
}

func (f *fakeAgentHandle) Messages() []openai.ChatCompletionMessage { return f.msgs }
func (f *fakeAgentHandle) SetMessages(msgs []openai.ChatCompletionMessage) {
	f.msgs = msgs
}
func (f *fakeAgentHandle) Config() *config.Config { return nil }
func (f *fakeAgentHandle) Reconfigure()           {}
func (f *fakeAgentHandle) Compact() error         { return nil }
func (f *fakeAgentHandle) Reset()                 {}

func executeWrite(t *testing.T, args string, msgs []openai.ChatCompletionMessage) *plugin.ChatCommandResponse {
	t.Helper()

	cmd := &Plugin{}
	params := &plugin.ChatCommandInput{
		AgentHandler: &fakeAgentHandle{msgs: msgs},
		Args:         args,
	}

	resp, err := cmd.execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}
	return resp
}

func assistantHistory(content string) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "question"},
		{Role: openai.ChatMessageRoleAssistant, Content: content},
	}
}

func TestUsageWhenArgsEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")

	resp := executeWrite(t, "", assistantHistory("answer"))

	if resp.Stdout != "Usage: `/write <filename>`\n" {
		t.Fatalf("Expected usage hint, got %q", resp.Stdout)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Expected no file to be written, stat err = %v", err)
	}
}

func TestWriteSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "answer.md")
	content := "# Answer\n\nSome **markdown** content.\n"

	resp := executeWrite(t, path, assistantHistory(content))

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be written, got %v", err)
	}
	if string(got) != content {
		t.Fatalf("Expected file content %q, got %q", content, string(got))
	}

	expected := fmt.Sprintf("## Write\n\n✅ Saved %d bytes to `%s`\n", len(content), path)
	if resp.Stdout != expected {
		t.Fatalf("Expected stdout %q, got %q", expected, resp.Stdout)
	}
}

func TestWriteSuccessFormatsMarkdownDiagnostic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	content := "plain text answer"

	resp := executeWrite(t, path, assistantHistory(content))

	if !strings.HasPrefix(resp.Stdout, "## Write\n\n") {
		t.Fatalf("Expected markdown heading, got %q", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "✅") {
		t.Fatalf("Expected success indicator, got %q", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "`"+path+"`") {
		t.Fatalf("Expected file path in code span, got %q", resp.Stdout)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("Expected trailing newline, got %q", resp.Stdout)
	}
}

func TestWriteFallsBackToPreviousMessage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "fallback.md")
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "question"},
		{Role: openai.ChatMessageRoleAssistant, Content: "the answer"},
		{Role: openai.ChatMessageRoleUser, Content: "follow-up"},
	}

	resp := executeWrite(t, path, msgs)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be written, got %v", err)
	}
	if string(got) != "the answer" {
		t.Fatalf("Expected previous message content, got %q", string(got))
	}
	if !strings.Contains(resp.Stdout, "✅") {
		t.Fatalf("Expected success indicator, got %q", resp.Stdout)
	}
}

func TestWritePicksLastMessageInLongerHistory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "long.md")
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: "system prompt"},
		{Role: openai.ChatMessageRoleUser, Content: "first question"},
		{Role: openai.ChatMessageRoleAssistant, Content: "first answer"},
		{Role: openai.ChatMessageRoleUser, Content: "second question"},
		{Role: openai.ChatMessageRoleAssistant, Content: "second answer"},
	}

	executeWrite(t, path, msgs)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be written, got %v", err)
	}
	if string(got) != "second answer" {
		t.Fatalf("Expected last assistant message, got %q", string(got))
	}
}

func TestWriteNoConversationHistory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "question"},
	}

	resp := executeWrite(t, path, msgs)

	expected := "## Write\n\n⚠️ Nothing to save: no conversation history\n"
	if resp.Stdout != expected {
		t.Fatalf("Expected stdout %q, got %q", expected, resp.Stdout)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Expected no file to be written, stat err = %v", err)
	}
}

func TestWriteEmptyAssistantMessage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")

	resp := executeWrite(t, path, assistantHistory(""))

	expected := "## Write\n\n⚠️ Nothing to save: last assistant message is empty\n"
	if resp.Stdout != expected {
		t.Fatalf("Expected stdout %q, got %q", expected, resp.Stdout)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Expected no file to be written, stat err = %v", err)
	}
}

func TestWriteFailureInvalidPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "missing-parent", "out.md")

	resp := executeWrite(t, path, assistantHistory("answer"))

	if !strings.HasPrefix(resp.Stdout, "## Write\n\n❌ Failed to save to `"+path+"`: ") {
		t.Fatalf("Expected failure diagnostic with path, got %q", resp.Stdout)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("Expected trailing newline, got %q", resp.Stdout)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Expected no file to be written, stat err = %v", err)
	}
}

func TestWriteFailurePathIsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	resp := executeWrite(t, dir, assistantHistory("answer"))

	if !strings.Contains(resp.Stdout, "❌ Failed to save to `"+dir+"`") {
		t.Fatalf("Expected failure diagnostic, got %q", resp.Stdout)
	}
}

func TestWriteSpecialCharacters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "special.md")
	content := "unicode: 한국어 ✨\nbackticks: ```code```\nquotes: \"double\" 'single'\n"

	executeWrite(t, path, assistantHistory(content))

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be written, got %v", err)
	}
	if string(got) != content {
		t.Fatalf("Expected byte-exact content, got %q", string(got))
	}
}

func TestWriteLargeContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.md")
	content := strings.Repeat("0123456789", 100000) // 1 MB

	resp := executeWrite(t, path, assistantHistory(content))

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected file to be written, got %v", err)
	}
	if len(got) != 1000000 {
		t.Fatalf("Expected 1000000 bytes, got %d", len(got))
	}
	if !strings.Contains(resp.Stdout, "Saved 1000000 bytes") {
		t.Fatalf("Expected byte count in diagnostic, got %q", resp.Stdout)
	}
}

func TestWriteExpandsHome(t *testing.T) {
	t.Parallel()

	usr, err := user.Current()
	if err != nil {
		t.Skipf("cannot resolve current user: %v", err)
	}
	dir, err := os.MkdirTemp(usr.HomeDir, ".magnetar-write-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "answer.md")

	resp := executeWrite(t, "~/"+filepath.Base(dir)+"/answer.md", assistantHistory("home answer"))

	if strings.HasPrefix(strings.TrimPrefix(resp.Stdout, "## Write\n\n"), "✅ Saved") {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Expected file in expanded home, got %v", err)
		}
		if string(got) != "home answer" {
			t.Fatalf("Expected file content %q, got %q", "home answer", string(got))
		}
		if !strings.Contains(resp.Stdout, "`"+path+"`") {
			t.Fatalf("Expected expanded path in diagnostic, got %q", resp.Stdout)
		}
		return
	}
	t.Fatalf("Expected success diagnostic, got %q", resp.Stdout)
}
