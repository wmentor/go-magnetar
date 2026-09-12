package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
)

// searchLimitPlugin registers one search tool that always succeeds and counts
// how many times it was actually executed.
type searchLimitPlugin struct {
	calls *int32
}

func (p searchLimitPlugin) Init(s *plugin.State, hub plugin.Hub) error {
	hub.RegisterTool(plugin.LLMTool{
		Definition: func() openai.Tool {
			return openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:       "fake_search",
					Parameters: map[string]any{"type": "object"},
				},
			}
		},
		Execute: func(ctx context.Context, args string) (string, error) {
			atomic.AddInt32(p.calls, 1)
			return "hit", nil
		},
		IsSearchTool: true,
	})
	return nil
}

var (
	searchLimitSetupOnce sync.Once
	searchToolCalls      int32
)

// registerSearchTool puts the fake search tool into the process-global registry.
// plugin.InitAll may only run once per process, hence the sync.Once.
func registerSearchTool(t *testing.T) {
	t.Helper()
	searchLimitSetupOnce.Do(func() {
		plugin.Register("search-limit-test", searchLimitPlugin{calls: &searchToolCalls})
		if err := plugin.InitAll(&plugin.State{}); err != nil {
			t.Fatalf("plugin.InitAll: %v", err)
		}
	})
}

// alwaysToolCallServer answers every chat completion with a tool call, even when
// the request offered no tools. This is the provider behaviour described in
// issue #15.
func alwaysToolCallServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 0,
			"model":   "test-model",
			"choices": []any{map[string]any{
				"index": 0,
				"message": map[string]any{
					"role": "assistant",
					"tool_calls": []any{map[string]any{
						"id":   "call_1",
						"type": "function",
						"function": map[string]any{
							"name":      "fake_search",
							"arguments": "{}",
						},
					}},
				},
				"finish_reason": "tool_calls",
			}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeTestConfig(t *testing.T, baseURL string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	yaml := "profile: default\n" +
		"language: english\n" +
		"profiles:\n" +
		"  default:\n" +
		"    llm:\n" +
		"      base_url: " + baseURL + "\n" +
		"      api_key: test-key\n" +
		"      model: test-model\n" +
		"      context: 128000\n" +
		"      temperature: 0.9\n" +
		"      top_p: 0.95\n" +
		"      reasoning_effort: high\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// TestAskStopsWhenTheModelIgnoresTheSearchLimit reproduces issue #15: after the
// search budget is exhausted the model keeps asking for tool calls, and before
// the fix Ask never returned.
func TestAskStopsWhenTheModelIgnoresTheSearchLimit(t *testing.T) {
	printer.SetDefault(printer.New(false))
	registerSearchTool(t)
	srv := alwaysToolCallServer(t)

	cfg, err := config.Load(writeTestConfig(t, srv.URL))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	atomic.StoreInt32(&searchToolCalls, 0)
	agent, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_, _ = agent.Ask("search something")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Ask never returned: the tool loop kept running after the search call limit (issue #15)")
	}

	if got := atomic.LoadInt32(&searchToolCalls); got != maxSearchToolCalls {
		t.Fatalf("search tool executed %d times, want exactly %d", got, maxSearchToolCalls)
	}
}
