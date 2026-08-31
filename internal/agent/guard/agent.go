package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

const (
	guardTimeout = 30 * time.Second
)

const guardPrompt = `You are a security guard that analyzes shell commands for safety.

Your task is to analyze the command and determine if it's safe to execute.

Return a JSON object with:
- "allowed": boolean (true if safe, false if dangerous)
- "reason": string explaining the decision

IMPORTANT: Return ONLY the raw JSON object. Do NOT wrap it in markdown code blocks (` + "```json ... ```" + `), do NOT add any explanations, do NOT add any additional text before or after the JSON object.

Security criteria to check:
- Destructive commands (rm -rf /, sudo, mkfs, dd with /dev/, fdisk, chmod 777)
- Shell pipes (| bash, | sh, | zsh)
- Git modifications (git commit, push, rebase, pull, cherry-pick, reset, stash, clean, reflog)
- Untrusted script execution (curl ... | bash, wget ... | sh)

Always check for danger patterns first. If the command looks dangerous, return allowed=false with a clear reason.

IMPORTANT: If you are uncertain about whether the command is safe, ALWAYS return allow=false with a reason explaining your concerns. Err on the side of caution - better to block a safe command than to allow a dangerous one.

Command to analyze:
{{COMMAND}}
`

type Guard struct {
	cfg *config.Config
	llm *openai.Client
}

type Response struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

func New(cfg *config.Config) *Guard {
	llmCfg := openai.DefaultConfig(cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = cfg.ProfileParamString("llm.base_url")
	return &Guard{
		cfg: cfg,
		llm: openai.NewClientWithConfig(llmCfg),
	}
}

func (g *Guard) CheckSecurity(command string, readOnly bool) (allowed bool, reason string, err error) {
	printer.ToolCall(printer.IconGuard, "guard: check command")

	ctx, cancel := context.WithTimeout(context.Background(), guardTimeout)
	defer cancel()

	prompt := strings.ReplaceAll(guardPrompt, "{{COMMAND}}", command)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompt,
		},
	}

	req := openai.ChatCompletionRequest{
		Model:    g.cfg.ProfileParamString("llm.model"),
		Messages: messages,
	}

	resp, err := g.llm.CreateChatCompletion(ctx, req)
	if err != nil {
		return false, fmt.Sprintf("LLM call failed: %v", err), err
	}

	if len(resp.Choices) == 0 {
		return false, "no response from LLM", fmt.Errorf("no choices in response")
	}

	content := resp.Choices[0].Message.Content

	var result Response
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return false, fmt.Sprintf("failed to parse LLM response: %v", err), err
	}

	return result.Allowed, result.Reason, nil
}
