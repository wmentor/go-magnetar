package ssh

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/melbahja/goph/v2"

	"github.com/wmentor/go-magnetar/internal/agent/guard"
	"github.com/wmentor/go-magnetar/internal/command/interceptor"
	"github.com/wmentor/go-magnetar/internal/common"
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
	return "ssh"
}

func (t *Tool) Description() string {
	return "Execute a shell command on a remote server via SSH. Connection parameters (user, port, key, " +
		"password, use_ssh_agent) are loaded from ssh.* config."
}

func (t *Tool) Exec(ctx context.Context, params *SSHParams) (any, error) {
	if params == nil {
		params = &SSHParams{}
	}

	t.printer.ToolCall(printer.IconTool, "ssh", "command", params.Command, "addr", params.Addr)

	if t.state.ReadOnly {
		t.printer.ToolCall(printer.IconBlocked, "the ssh tool is forbidden in read-only mode", "command", params.Command)
		return nil, fmt.Errorf("the ssh tool is forbidden in read-only mode")
	}

	if !t.cfg.Bool("ssh.enable") {
		t.printer.Print(printer.IconBlocked, "ssh: disabled")
		return nil, fmt.Errorf("ssh disabled")
	}

	allowed, reason := t.checkSecurity(params.Command)
	if !allowed {
		t.printer.Print(printer.IconBlocked, "ssh: blocked dangerous command", "command", params.Command, "reason", reason)
		return nil, fmt.Errorf("ssh blocked: %s", reason)
	}

	if params.Addr == "" {
		t.printer.Error("ssh: no address provided")
		return nil, fmt.Errorf("no address provided")
	}

	sshUser := t.cfg.String("ssh.user")
	if params.User != "" {
		sshUser = strings.TrimSpace(params.User)
	}
	if sshUser == "" {
		usr, err := user.Current()
		if err != nil {
			t.printer.Error("ssh: failed to get current user", "err", err)
			return nil, fmt.Errorf("failed to get current user")
		}
		sshUser = usr.Username
	}

	remoteDir := t.cfg.String("ssh.remote_dir")
	if params.Dir != "" {
		remoteDir = strings.TrimSpace(params.Dir)
	}
	if remoteDir == "" {
		remoteDir = "$HOME"
	}

	timeout := t.cfg.Int("ssh.timeout")
	if timeout <= 0 {
		timeout = 120
	}

	var opts []goph.Option
	if key := t.cfg.String("ssh.key"); key != "" {
		key = common.ExpandHome(key)
		opts = append(opts, goph.WithKeyFile(key, ""))
	} else if password := t.cfg.String("ssh.password"); password != "" {
		opts = append(opts, goph.WithPassword(password))
	} else if t.cfg.Bool("ssh.use_ssh_agent") {
		opts = append(opts, goph.WithDefaultAgent())
	} else {
		t.printer.Error("ssh: no authentication method configured")
		return nil, fmt.Errorf("no authentication method configured")
	}

	opts = append(opts, goph.WithTimeout(time.Duration(timeout)*time.Second), goph.WithInsecureIgnoreHostKey())

	client, err := goph.New(sshUser, params.Addr, opts...)
	if err != nil {
		t.printer.Error("ssh: connection failed", "addr", params.Addr, "err", err)
		return nil, fmt.Errorf("ssh connection failed: %w", err)
	}
	defer client.Close()

	cmdStr := params.Command
	if remoteDir != "" {
		cmdStr = "cd " + remoteDir + " && " + params.Command
	}
	cmd, err := client.Command("sh", "-c", cmdStr)
	if err != nil {
		t.printer.Error("ssh: failed to create command", "command", params.Command, "err", err)
		return nil, fmt.Errorf("failed to create SSH command: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.printer.Error("ssh: command execution failed", "command", params.Command, "err", err)
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	outputStr := string(output)
	if len(output) > execOutputMaxSize {
		t.printer.Print(printer.IconAlert, "ssh truncated output", "command", params.Command)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: outputStr[:execOutputMaxSize] + fmt.Sprintf("\n(output truncated to %d bytes)", execOutputMaxSize),
		}, nil
	}

	return &tools.Response{
		Status: tools.StatusOK,
		Result: outputStr,
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

type SSHParams struct {
	Command string `json:"command" jsonschema_description:"The shell command to execute"`
	Stdin   string `json:"stdin,omitempty" jsonschema_description:"Standard input to pass to the command (optional)"`
	Addr    string `json:"addr" jsonschema_description:"Remote server address (hostname or hostname:port, required)"`
	User    string `json:"user,omitempty" jsonschema_description:"SSH username (optional, if not provided uses ssh.user from config)"`
	Dir     string `json:"dir,omitempty" jsonschema_description:"Remote working directory (optional, if not provided uses ssh.remote_dir from config)"`
}
