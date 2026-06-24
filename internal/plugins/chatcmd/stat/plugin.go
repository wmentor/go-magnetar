package statplugin

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/go-units"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.stat", &Plugin{})
}

// Plugin registers the /stat chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "stat",
		Help: "Print context statistics (messages, tokens, bytes, models).",
		Execute: func(_ context.Context, a plugin.AgentHandle, _ string) error {
			msgs := a.Messages()
			cfg := a.Config()

			var totalBytes, totalTokens int
			for _, m := range msgs {
				b := len(m.Role) + len(m.Content)
				for _, tc := range m.ToolCalls {
					b += len(tc.Function.Name) + len(tc.Function.Arguments)
				}
				totalBytes += b
				t := b / 4
				if t < 1 {
					t = 1
				}
				totalTokens += t
			}

			fmt.Fprintf(os.Stdout,
				"Context stats:\n  messages    : %d\n  tokens      : ~%d (estimated)\n  bytes       : %s\n  llm model   : %s\n  rag model   : %s\n  vector size : %d\n\n",
				len(msgs),
				totalTokens,
				units.HumanSize(float64(totalBytes)),
				cfg.LLM.Model,
				cfg.RAG.LLM.Model,
				cfg.RAG.LLM.VectorSize,
			)
			return nil
		},
	})
	return nil
}

// ensure ToolCalls field is accessible — import used for type reference only.
var _ = openai.ChatCompletionMessage{}
