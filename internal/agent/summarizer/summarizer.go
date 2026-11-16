package summarizer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const (
	// ModeSoft preserves the trailing save_tail messages unchanged and summarizes the earlier history.
	ModeSoft = "soft"
	// ModeCompact summarizes the entire non-system conversation history into a single message.
	ModeCompact = "compact"
	// ModeFull is an alias for ModeCompact (full history compaction).
	ModeFull = "full"

	summaryPrompt = `You are a conversation summarizer. You will be given a sequence of chat messages.
Produce a concise summary that preserves all factual details, decisions, and context needed to continue the conversation.
Write the summary as a single assistant message in the same language as the conversation.
Do not add any preamble or meta-commentary — output only the summary text.`
)

// Summarizer condenses a slice of chat messages into a single summary message
// using the configured LLM.
type Summarizer struct {
	cfg *config.Config
	llm *openai.Client
}

// New creates a Summarizer backed by the same LLM as the chat agent.
func New(cfg *config.Config) *Summarizer {
	llmCfg := openai.DefaultConfig(cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = cfg.ProfileParamString("llm.base_url")
	llmCfg.HTTPClient = requester.NewFactory(cfg).Get()
	return &Summarizer{
		cfg: cfg,
		llm: openai.NewClientWithConfig(llmCfg),
	}
}

// threshold returns the effective token threshold for triggering compaction.
// If compact.threshold is not set (≤ 0), defaults to 80 % of llm.context.
func (s *Summarizer) threshold() int {
	if cfgThreshold := s.cfg.Int("compact.threshold"); cfgThreshold > 0 {
		return cfgThreshold
	}
	if cfgContext := s.cfg.ProfileParamInt("llm.context"); cfgContext > 0 {
		return cfgContext * 4 / 5
	}
	return 0 // no limit configured
}

// NeedsCompaction reports whether the total estimated token count of messages
// exceeds the configured threshold.
func (s *Summarizer) NeedsCompaction(messages []openai.ChatCompletionMessage) bool {
	t := s.threshold()
	if t <= 0 {
		return false
	}
	total := 0
	for _, m := range messages {
		total += estimateTokens(m)
	}
	return total >= t
}

// Compact compresses the conversation history according to the configured
// compact.mode strategy ("soft" by default, or "compact"/"full"), or an
// explicit mode override ("soft", "full", or "compact") when provided.
//
// The system message (index 0) is always preserved verbatim and is NOT passed
// to the summarizer — only non-system messages are condensed.
func (s *Summarizer) Compact(messages []openai.ChatCompletionMessage, mode ...string) ([]openai.ChatCompletionMessage, error) {
	if len(messages) <= 1 {
		return messages, nil
	}

	effectiveMode := s.Mode()
	if len(mode) > 0 && strings.TrimSpace(mode[0]) != "" {
		switch strings.ToLower(strings.TrimSpace(mode[0])) {
		case ModeCompact, ModeFull:
			effectiveMode = ModeCompact
		case ModeSoft:
			effectiveMode = ModeSoft
		default:
			return nil, fmt.Errorf("summarizer: unknown compact mode %q (expected %q or %q)", mode[0], ModeSoft, ModeFull)
		}
	}

	switch effectiveMode {
	case ModeCompact:
		return s.compactFull(messages)
	default:
		return s.compactSoft(messages)
	}
}

// compactFull implements the "compact" strategy:
//  1. Reads the full conversation history (all non-system messages)
//  2. Uses the LLM to generate a summary of the entire conversation
//  3. Replaces the entire conversation history with a single summary message
//     following the preserved system prompt: [system, summary-assistant-msg].
func (s *Summarizer) compactFull(messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if len(messages) <= 1 {
		return messages, nil
	}
	system := messages[0]
	rest := messages[1:]

	printer.Info("summarizer: compacting full history",
		"mode", ModeCompact,
		"summarize", len(rest),
	)

	summary, err := s.summarize(rest)
	if err != nil {
		return nil, fmt.Errorf("summarizer: %w", err)
	}

	result := []openai.ChatCompletionMessage{
		system,
		{
			Role:    openai.ChatMessageRoleAssistant,
			Content: summary,
		},
	}

	printer.Debug("summarizer: history compacted",
		"mode", ModeCompact,
		"before", len(messages),
		"after", len(result),
	)

	return result, nil
}

// compactSoft implements the "soft" strategy: it summarizes the «head» portion
// of messages (everything except the last save_tail messages) and returns:
//
//	[system, summary-assistant-msg, ...tail]
//
// If save_tail < 1 all non-system messages are summarized.
func (s *Summarizer) compactSoft(messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	if len(messages) <= 1 {
		return messages, nil
	}
	// Separate system message from the rest.
	system := messages[0]
	rest := messages[1:]

	saveTail := s.cfg.Int("compact.save_tail")
	if saveTail < 1 || saveTail >= len(rest) {
		saveTail = 0
	}

	toSummarize := rest
	tail := []openai.ChatCompletionMessage{}
	if saveTail > 0 {
		toSummarize = rest[:len(rest)-saveTail]
		tail = rest[len(rest)-saveTail:]
	}

	if len(toSummarize) == 0 {
		// Nothing to summarize (tail covers everything).
		return messages, nil
	}

	printer.Info("summarizer: compacting history",
		"mode", ModeSoft,
		"summarize", len(toSummarize),
		"preserve_tail", len(tail),
	)

	summary, err := s.summarize(toSummarize)
	if err != nil {
		return nil, fmt.Errorf("summarizer: %w", err)
	}

	result := make([]openai.ChatCompletionMessage, 0, 2+len(tail))
	result = append(result, system)
	result = append(result, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: summary,
	})
	result = append(result, tail...)

	printer.Debug("summarizer: history compacted",
		"mode", ModeSoft,
		"before", 1+len(rest),
		"after", len(result),
	)

	return result, nil
}

// summarize calls the LLM to produce a textual summary of the given messages.
func (s *Summarizer) summarize(messages []openai.ChatCompletionMessage) (string, error) {
	// Build a plain-text transcript to pass to the summarizer.
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(m.Role)
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}

	req := openai.ChatCompletionRequest{
		Model: s.cfg.ProfileParamString("llm.model"),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: summaryPrompt},
			{Role: openai.ChatMessageRoleUser, Content: sb.String()},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	resp, err := s.llm.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM summarization request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM during summarization")
	}

	return resp.Choices[0].Message.Content, nil
}

// estimateTokens returns a rough token estimate for a single message.
// 1 token ≈ 4 characters (matches the approximation used in the chat agent).
func estimateTokens(m openai.ChatCompletionMessage) int {
	chars := len(m.Role) + len(m.Content)
	for _, tc := range m.ToolCalls {
		chars += len(tc.Function.Name) + len(tc.Function.Arguments)
	}
	t := max(chars/4, 1)
	return t
}

// Mode returns the configured compaction strategy ("soft" or "compact").
// Defaults to ModeSoft ("soft") when unset or unrecognized.
func (s *Summarizer) Mode() string {
	if s.cfg != nil {
		switch strings.ToLower(strings.TrimSpace(s.cfg.String("compact.mode"))) {
		case ModeCompact, ModeFull:
			return ModeCompact
		}
	}
	return ModeSoft
}
