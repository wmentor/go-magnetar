package execplugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.exec", &Plugin{})
}

type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "exec",
		Aliases: []string{"e"},
		Help:    "Execute a shell command. Usage: /exec <command> [args...]",
		Execute: p.execute,
	})
	return nil
}

func (p *Plugin) execute(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	args := strings.TrimSpace(params.Args)
	if args == "" {
		fmt.Fprintln(os.Stdout, "Usage: /exec <command> [args...]")
		return plugin.NopChatCommandResponse, nil
	}

	// security: block dangerous patterns
	if isDangerousCommand(args) {
		fmt.Fprintln(os.Stdout, "Error: command blocked by security policy")
		return plugin.NopChatCommandResponse, nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", args)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// Print stdout/stderr to stdout
	if stdoutStr != "" {
		fmt.Fprint(os.Stdout, stdoutStr)
	}
	if stderrStr != "" {
		fmt.Fprint(os.Stdout, stderrStr)
	}

	// Print exit code if command failed
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			fmt.Fprintf(os.Stdout, "\n[Exit code: %d]\n", exitErr.ExitCode())
		} else {
			fmt.Fprintf(os.Stdout, "\n[Error: %v]\n", err)
		}
	}

	return plugin.NopChatCommandResponse, nil
}

// isDangerousCommand checks if the command matches any dangerous pattern.
func isDangerousCommand(cmd string) bool {
	lower := strings.ToLower(cmd)

	// Block dangerous patterns using regex for more complex matching
	dangerousPatterns := []*regexp.Regexp{
		// Delete commands
		regexp.MustCompile(`(?i)\brm\s+-rf\s+[/$]|^rm\s+-rf\s+\*`),
		// Disk manipulation
		regexp.MustCompile(`(?i)\bdd\b.*\bif=\s*`), // dd with if= parameter
		regexp.MustCompile(`(?i)\bmkfs\b`),
		regexp.MustCompile(`(?i)\bfdisk\b`),
		// Shell execution
		regexp.MustCompile(`(?i)\bbash\s+-c\b`),
		regexp.MustCompile(`(?i)\bsh\s+-c\b`),
		regexp.MustCompile(`(?i)\bzsh\s+-c\b`),
		// Pipe to shell
		regexp.MustCompile(`(?i)(curl|wget)\b.*\|\s*(bash|sh|zsh)\b`),
	}

	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}

	return false
}
