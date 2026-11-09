package readplugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wmentor/go-magnetar/internal/common"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/tools"
	"github.com/wmentor/go-magnetar/internal/tools/file/read"
)

func init() {
	plugin.Register("chatcmd.read", &Plugin{})
}

const (
	timeout = time.Second * 30
)

var fromTest = false

type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s

	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "read",
		Help:    "Read the contents of a file. Usage: /read <filename>",
		Execute: p.execute,
	})

	return nil
}

func (p *Plugin) execute(ctx context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := params.Args
	if args == "" {
		return &plugin.ChatCommandResponse{Stdout: "Usage: /read <filename>"}, nil
	}

	filename := common.ExpandHome(filepath.Clean(args))

	// Create a new tool instance with nil root for unrestricted file access
	tool := read.New(p.state.Config, p.state.Printer, p.state.RequesterFactory, nil, p.state)

	content, err := tool.Exec(ctx, &read.FileReadParams{Filename: filename})
	if err != nil {
		return &plugin.ChatCommandResponse{Stdout: fmt.Sprintf("Error reading file %s: %v\n", filename, err)}, nil
	}

	// Extract the result from the tools.Response struct if needed
	var text string
	switch v := content.(type) {
	case string:
		text = v
	case *tools.Response:
		text = v.Result
	default:
		return &plugin.ChatCommandResponse{Stdout: "Error: unexpected response type from file_read tool\n"}, nil
	}

	var result string

	if p.checkLess() {
		// Use less for interactive viewing
		stdin := bytes.NewBuffer(nil)
		stdin.WriteString(text)

		cmd := exec.CommandContext(ctx, "less")
		cmd.Stdin = stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			result = fmt.Sprintf("Error running less: %v\n%s", err, text)
		}
		p.state.Printer.EmptyLine()
	} else {
		// Collect output for test mode
		result = text
	}

	return &plugin.ChatCommandResponse{Stdout: result}, nil
}

func (p *Plugin) checkLess() bool {
	if fromTest {
		return false
	}
	_, err := exec.LookPath("less")
	return err == nil
}
