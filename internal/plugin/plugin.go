package plugin

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
)

// ErrExit is a sentinel error that a ChatCommand.Execute function may return
// to signal the REPL loop to terminate the session.
var ErrExit = errors.New("exit")

// Plugin is the only interface a plugin package must implement.
type Plugin interface {
	// Init is called once after the config is loaded.
	// The plugin registers its contributions through hub and initialises
	// any internal state from s.
	Init(s *State, hub Hub) error
}

// State carries shared infrastructure injected into every plugin at Init time.
// Named State (not Context) to avoid confusion with context.Context.
type State struct {
	Config *config.Config
	Root   *os.Root     // sandboxed filesystem; nil in the indexer context
	Log    *slog.Logger
}

// Hub is the registration and lifecycle interface passed to Plugin.Init.
// A plugin calls the Register* methods to contribute functionality,
// and Go to enqueue managed background goroutines.
type Hub interface {
	// RegisterTool adds an LLM tool to the agent tool-use loop.
	RegisterTool(tool LLMTool)

	// RegisterChatCommand adds a slash command to the chat REPL.
	RegisterChatCommand(cmd ChatCommand)

	// RegisterCLICommand adds a subcommand to the CLI.
	// cmd must be a pointer to a kong-annotated struct with a Run() method.
	RegisterCLICommand(cmd any)

	// Go enqueues a background goroutine to be started after all plugins
	// have finished Init (i.e. after InitAll returns).
	// Each goroutine receives a context.Context that is cancelled when
	// Hub.Stop() is called. Hub.Stop() blocks until all goroutines return.
	Go(f func(ctx context.Context))

	// Stop cancels all goroutine contexts and waits for them to finish.
	// Called once by main when the program is shutting down.
	Stop()
}

// LLMTool is a single tool exposed to the LLM in the tool-use loop.
type LLMTool struct {
	// Definition returns the OpenAI-compatible tool schema.
	Definition func() openai.Tool
	// Execute is called when the LLM invokes this tool.
	// args is the raw JSON arguments string.
	Execute func(ctx context.Context, args string) (string, error)
	// IsSearchTool marks tools that count toward the per-request search call
	// limit (e.g. rag_search, web_fetch).
	IsSearchTool bool
}

// ChatCommand is a slash command available in the chat REPL.
type ChatCommand struct {
	// Name is the primary command name without "/", e.g. "stat".
	// Matching is case-insensitive.
	Name    string
	Aliases []string
	// Help is a one-line description shown in /help output.
	Help string
	// Execute is called when the command is matched.
	// args contains any text following the command name (trimmed).
	Execute func(ctx context.Context, agent AgentHandle, args string) error
}

// AgentHandle is passed to ChatCommand.Execute.
// It exposes the minimal API that slash commands need without giving
// plugins direct access to ChatAgent internals.
type AgentHandle interface {
	// Messages returns a read-only snapshot of conversation history.
	Messages() []openai.ChatCompletionMessage
	// SetMessages replaces conversation history.
	SetMessages([]openai.ChatCompletionMessage)
	// Config returns the agent's effective configuration.
	Config() *config.Config
	// Compact runs history compression via the summarizer and replaces
	// the history with the compacted version.
	Compact() error
	// Reset clears conversation history, keeping only the system prompt.
	Reset()
}
