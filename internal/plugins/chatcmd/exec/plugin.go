package execplugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/wmentor/go-magnetar/internal/command/interceptor"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.exec", &Plugin{})
}

type Plugin struct {
	state       *plugin.State
	interceptor *interceptor.Interceptor
}

func (p *Plugin) Init(state *plugin.State, hub plugin.Hub) error {
	p.state = state
	p.interceptor = interceptor.New()
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
	if p.interceptor.IsDangerous(args) {
		fmt.Fprintln(os.Stdout, "Error: command blocked by security policy")
		return plugin.NopChatCommandResponse, nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", args)

	buffer := bytes.NewBuffer(nil)

	sout := io.MultiWriter(os.Stdout, buffer)
	serr := io.MultiWriter(os.Stderr, buffer)

	cmd.Stdout = sout
	cmd.Stderr = serr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			fmt.Fprintf(sout, "\n[Exit code: %d]\n", exitErr.ExitCode())
		} else {
			fmt.Fprintf(sout, "\n[Error: %v]\n", err)
		}
	}

	p.state.ResponseStore.Push(buffer.String())

	return plugin.NopChatCommandResponse, nil
}
