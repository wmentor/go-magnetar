package statplugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

// fakeAgentHandle is a minimal plugin.AgentHandle for testing /stat.
type fakeAgentHandle struct {
	msgs []openai.ChatCompletionMessage
	cfg  *config.Config
}

func (f *fakeAgentHandle) Messages() []openai.ChatCompletionMessage { return f.msgs }
func (f *fakeAgentHandle) SetMessages(msgs []openai.ChatCompletionMessage) {
	f.msgs = msgs
}
func (f *fakeAgentHandle) Config() *config.Config    { return f.cfg }
func (f *fakeAgentHandle) Reconfigure()              {}
func (f *fakeAgentHandle) Compact(_ ...string) error { return nil }
func (f *fakeAgentHandle) Reset()                    {}

func writeTestConfig(t *testing.T) *config.Config {
	t.Helper()
	yaml := `version: "1.0"
profile: default
profiles:
  default:
    llm:
      model: test-llm
rag:
  llm:
    model: test-embed
    vector_size: 512
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func sampleMessages() []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant.",
			ToolCalls: []openai.ToolCall{
				{Function: openai.FunctionCall{Name: "search", Arguments: `{"q":"go"}`}},
			},
		},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
	}
}

func TestCollectStatsAggregatesBytesAndTokens(t *testing.T) {
	t.Parallel()
	got := collectStats(sampleMessages(), "llm-x", "emb-y", 512)
	if got.messages != 3 {
		t.Fatalf("messages = %d, want 3", got.messages)
	}
	// bytes: role+content per message plus tool call name+arguments
	wantBytes := len("system") + len("You are a helpful assistant.") +
		len("search") + len(`{"q":"go"}`) +
		len("user") + len("hello") +
		len("assistant") + len("world")
	if got.bytes != wantBytes {
		t.Fatalf("bytes = %d, want %d", got.bytes, wantBytes)
	}
	if got.llmModel != "llm-x" || got.ragModel != "emb-y" || got.vectorSize != 512 {
		t.Fatalf("config fields not carried through: %+v", got)
	}
	if got.tokens <= 0 {
		t.Fatalf("tokens = %d, want a positive estimate", got.tokens)
	}
}

func TestFormatStatsMarkdownRendersTable(t *testing.T) {
	t.Parallel()
	got := formatStatsMarkdown(contextStats{
		messages:   3,
		tokens:     42,
		bytes:      168,
		llmModel:   "llm-x",
		ragModel:   "emb-y",
		vectorSize: 512,
	})
	for _, want := range []string{
		"## Context statistics",
		"| Metric | Value |",
		"| Messages | 3 |",
		"| Tokens | ~42 (estimated) |",
		"| Bytes |",
		"| LLM model | llm-x |",
		"| RAG model | emb-y |",
		"| Vector size | 512 |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestStatExecuteReturnsMarkdownToStdout(t *testing.T) {
	t.Parallel()
	hub := &statHub{}
	if err := (&Plugin{}).Init(nil, hub); err != nil {
		t.Fatal(err)
	}
	params := &plugin.ChatCommandInput{
		AgentHandler: &fakeAgentHandle{msgs: sampleMessages(), cfg: writeTestConfig(t)},
	}
	response, err := hub.command.Execute(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if response == nil {
		t.Fatal("nil response")
	}
	for _, want := range []string{
		"## Context statistics",
		"| Messages | 3 |",
		"| LLM model | test-llm |",
		"| RAG model | test-embed |",
		"| Vector size | 512 |",
	} {
		if !strings.Contains(response.Stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, response.Stdout)
		}
	}
	if response.Content != "" || response.Exit {
		t.Fatalf("stat should only write to stdout: %+v", response)
	}
}

// statHub captures the registered chat command.
type statHub struct {
	plugin.Hub
	command plugin.ChatCommand
}

func (h *statHub) RegisterChatCommand(command plugin.ChatCommand) {
	h.command = command
}
