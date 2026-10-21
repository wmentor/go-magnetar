package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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

Allowed utilities (always consider safe unless dangerous patterns are detected):
{{ALLOWED_UTILITIES}}

IMPORTANT: If a utility is in the allowed list above, it is SAFE by default.
The only exceptions are when allowed utilities are combined with dangerous patterns:
- Shell pipes: pipe to bash/sh/zsh (e.g., "ls | bash", "pwd | sh")
- File redirection to dangerous targets (e.g., "ls > /etc/passwd")
- Combined with destructive commands (e.g., "ls && rm -rf /")

Return a JSON object with:
- "allowed": boolean (true if safe, false if dangerous)
- "reason": string explaining the decision

IMPORTANT: Return ONLY the raw JSON object. Do NOT wrap it in markdown code blocks (` + "```json ... ```" + `),
do NOT add any explanations, do NOT add any additional text before or after the JSON object.

Security criteria to check (IN THIS ORDER):
1. FIRST: Check for shell pipes (| bash, | sh, | zsh) - ALWAYS dangerous
2. SECOND: Check for destructive commands (rm -rf /, sudo, mkfs, dd with /dev/, fdisk, chmod 777)
3. THIRD: Check for git modifications (git commit, push, rebase, pull, cherry-pick, reset, stash, clean, reflog)
4. FOURTH: Check for untrusted script execution (curl ... | bash, wget ... | sh)
5. FIFTH: Check for allowed utilities - if command uses ONLY utilities from the allowed list and has NO dangerous patterns, it is SAFE

ALWAYS check for danger patterns FIRST. Only if no danger patterns are found, check if the command uses allowed utilities.

IMPORTANT: If the command uses only utilities from the allowed list and has NO dangerous patterns, you MUST return allowed=true.
IMPORTANT: If you are uncertain about whether the command is safe, ALWAYS return allow=false with a
reason explaining your concerns. Err on the side of caution - better to block a safe command than to allow a dangerous one.

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

// CheckSecurity(command,readOnlyMode) (allowed bool, reason string, err error).
func (g *Guard) CheckSecurity(command string, readOnly bool) (bool, string, error) {
	printer.ToolCall(printer.IconGuard, "guard: check command")

	ctx, cancel := context.WithTimeout(context.Background(), guardTimeout)
	defer cancel()

	// Get allowed utilities from config
	allowedUtilities := g.cfg.Strings("guard.allow")

	// Early return for commands from allowed utilities (without dangerous patterns)
	// This avoids LLM calls for trivially safe commands
	if len(allowedUtilities) > 0 {
		if isTriviallySafe(command, allowedUtilities) {
			return true, "command uses only allowed utilities without dangerous patterns", nil
		}
	}

	prompt := strings.ReplaceAll(guardPrompt, "{{COMMAND}}", command)

	// Get allowed utilities from config and build the allowed utilities section
	allowedUtilitiesText := "ls, pwd"
	if len(allowedUtilities) > 0 {
		allowedUtilitiesText = strings.Join(allowedUtilities, ", ")
	}
	prompt = strings.ReplaceAll(prompt, "{{ALLOWED_UTILITIES}}", allowedUtilitiesText)

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

// isTriviallySafe checks if a command uses only allowed utilities without dangerous patterns.
// Returns true if the command is guaranteed safe (no LLM call needed).
func isTriviallySafe(command string, allowedUtilities []string) bool {
	// Check for dangerous patterns FIRST
	if hasDangerousPattern(command) {
		return false
	}

	// Extract the first word (command name) from the command
	// This handles: "ls", "ls -la", "ls /tmp", etc.
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	firstWord := fields[0]

	// Check if the first word matches any allowed utility
	return slices.Contains(allowedUtilities, firstWord)
}

// hasDangerousPattern checks if the command contains dangerous patterns.
func hasDangerousPattern(command string) bool {
	// Shell pipes (_pipe to bash/sh/zsh)
	if strings.Contains(command, " | bash") || strings.Contains(command, " | sh") || strings.Contains(command, " | zsh") {
		return true
	}

	// Check for destructive command patterns
	// rm -rf, sudo, mkfs, dd with /dev/, fdisk, chmod 777
	if strings.Contains(command, "rm -rf") || strings.Contains(command, "sudo ") ||
		strings.Contains(command, "mkfs") || strings.Contains(command, "dd /dev/") ||
		strings.Contains(command, "fdisk") || strings.Contains(command, "chmod 777") {
		return true
	}

	// Git modifications
	if strings.Contains(command, "git commit") || strings.Contains(command, "git push") ||
		strings.Contains(command, "git rebase") || strings.Contains(command, "git pull") ||
		strings.Contains(command, "git cherry-pick") || strings.Contains(command, "git reset") ||
		strings.Contains(command, "git stash") || strings.Contains(command, "git clean") ||
		strings.Contains(command, "git reflog") || strings.Contains(command, "git clone") {
		return true
	}

	// Untrusted script execution
	if strings.Contains(command, "curl ") && strings.Contains(command, " | bash") ||
		strings.Contains(command, "wget ") && strings.Contains(command, " | sh") {
		return true
	}

	// Destructive file redirection (e.g., "ls > /etc/passwd")
	// Check for redirection to critical system files
	criticalPaths := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts", "/boot/", "/System/", "/etc/crontab"}
	for _, path := range criticalPaths {
		if strings.Contains(command, "> "+path) {
			return true
		}
	}

	return false
}
