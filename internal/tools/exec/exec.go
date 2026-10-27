package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/wmentor/go-magnetar/internal/agent/guard"
	"github.com/wmentor/go-magnetar/internal/command/interceptor"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const (
	execTimeout       = 1 * time.Minute
	execOutputMaxSize = 64 * 1024
)

type fixedSizeWriter struct {
	data    []byte
	written int
	limit   int
	full    bool
}

func newFixedSizeWriter(limit int) *fixedSizeWriter {
	return &fixedSizeWriter{limit: limit, data: make([]byte, limit)}
}

func (w *fixedSizeWriter) Write(p []byte) (int, error) {
	remain := w.limit - w.written
	if len(p) > remain {
		p = p[:remain]
		w.full = true
	}

	n := copy(w.data[w.written:], p)
	w.written += n
	return n, nil
}

func (w *fixedSizeWriter) String() string {
	return string(w.data[:w.written])
}

func (w *fixedSizeWriter) Truncated() bool {
	return w.full
}

type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	root          *os.Root
	state         *plugin.State
	guard         *guard.Guard
	interceptor   *interceptor.Interceptor
}

func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, root *os.Root, state *plugin.State) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		root:          root,
		state:         state,
		guard:         guard.New(cfg),
		interceptor:   interceptor.New(),
	}
}

func (t *Tool) Name() string {
	return "exec"
}

func (t *Tool) Description() string {
	return "Execute a shell command via \"sh -c\". The command runs with a clean environment " +
		"and the current working directory. Stdin can be provided for input."
}

func (t *Tool) Exec(ctx context.Context, params *ExecParams) (any, error) {
	if params == nil {
		params = &ExecParams{}
	}

	t.printer.ToolCall(printer.IconTool, "exec", "command", params.Command)

	if t.state.ReadOnly {
		t.printer.ToolCall(printer.IconBlocked, "the exec tool is forbidden in read-only mode", "command", params.Command)
		return nil, fmt.Errorf("the exec tool is forbidden in read-only mode")
	}

	allowed, reason := t.checkSecurity(params.Command)
	if !allowed {
		t.printer.Print(printer.IconBlocked, "exec: blocked dangerous command", "command", params.Command, "reason", reason)
		return nil, fmt.Errorf("exec blocked: %s", reason)
	}

	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	// #nosec G204 -- command validation is done in checkSecurity
	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)
	cmd.Stdin = strings.NewReader(params.Stdin)
	cmd.Dir = t.root.Name()

	cmd.Env = makeEnvs()

	w := newFixedSizeWriter(execOutputMaxSize)
	cmd.Stdout = w
	cmd.Stderr = w

	cmdErr := cmd.Run()
	if cmdErr != nil {
		t.printer.Error("exec: command failed", "command", params.Command, "error", cmdErr)
		if w.Truncated() {
			return nil, fmt.Errorf("exec command failed: %w (output truncated to %d bytes)", cmdErr, execOutputMaxSize)
		}
		return nil, fmt.Errorf("exec command failed: %w\n%s", cmdErr, w.String())
	}

	if w.Truncated() {
		t.printer.Print(printer.IconAlert, "exec truncated output", "command", params.Command)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: w.String() + fmt.Sprintf("\n(output truncated to %d bytes)", execOutputMaxSize),
		}, nil
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: w.String(),
	}, nil
}

func (t *Tool) checkSecurity(command string) (bool, string) {
	if t.isCommandBlocked(command) {
		return false, "blocked by security patterns"
	}

	if t.cfg.Bool("guard.enable") {
		allowed, reason, err := t.guard.CheckSecurity(command, t.state.ReadOnly)
		if err != nil {
			return false, fmt.Sprintf("security check failed: %v", err)
		}
		if !allowed {
			if t.cfg.Bool("guard.ask") {
				if askUserForCommand(command, reason) {
					return true, ""
				}
				return false, "user declined command execution"
			}
			return false, reason
		}
	}

	return true, ""
}

func (t *Tool) isCommandBlocked(command string) bool {
	return t.interceptor.IsBlocked(command)
}

// askUserForCommand prompts the user for confirmation before executing a blocked command.
func askUserForCommand(command, reason string) bool {
	fmt.Printf("Security guard blocked this command:\n\n")
	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Reason: %s\n\n", reason)

	var answer string

	for {
		fmt.Printf("Do you want to execute it anyway? (y/N): ")
		answer = ""
		_, _ = fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "y" || answer == "n" {
			break
		}
	}

	return answer == "y"
}

func makeEnvs() []string {
	envs := []string{}
	for _, rec := range os.Environ() {
		pair := strings.SplitN(rec, "=", 2)
		key := pair[0]

		if envFilter.MatchString(key) {
			continue
		}
		envs = append(envs, rec)
	}
	return envs
}

var envFilter = regexp.MustCompile(`(?i)(pass|secr|cred|token|key|auth|pwd|cert|sign|salt|bearer|amqp|connect)`)

type ExecParams struct {
	Command string `json:"command" jsonschema_description:"The shell command to execute"`
	Stdin   string `json:"stdin,omitempty" jsonschema_description:"Standard input to pass to the command (optional)"`
}
