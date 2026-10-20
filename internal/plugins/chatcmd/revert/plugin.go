package revertplugin

import (
	"context"
	"slices"

	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.revert", &Plugin{})
}

// Plugin registers the /revert chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "revert",
		Aliases: []string{"r"},
		Help:    "Revert to the previous assistant answer and display it again.",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			messages := params.AgentHandler.Messages()
			if len(messages) < 3 {
				return &plugin.ChatCommandResponse{
					Stdout: "Nothing to revert: not enough conversation history",
				}, nil
			}

			// Find the last assistant answer (without tool calls)
			// and the user message that preceded it.
			lastUserIndex := -1
			lastAssistantIndex := -1

			// Iterate from the end to find the last user-assistant pair
			for i, msg := range slices.Backward(messages) {
				if msg.Role == openai.ChatMessageRoleUser && msg.Content != "" {
					if lastUserIndex == -1 {
						lastUserIndex = i
						continue
					}
					// Found the second user message, so the last assistant message is between them
					for j := i + 1; j < lastUserIndex; j++ {
						if messages[j].Role == openai.ChatMessageRoleAssistant && len(messages[j].ToolCalls) == 0 {
							lastAssistantIndex = j
							break
						}
					}
					break
				}
			}

			// If we didn't find a user message before the last one,
			// check if the last message is user and find the previous assistant
			if lastAssistantIndex == -1 && lastUserIndex == len(messages)-1 {
				for i := len(messages) - 2; i >= 0; i-- {
					if messages[i].Role == openai.ChatMessageRoleAssistant && len(messages[i].ToolCalls) == 0 {
						lastAssistantIndex = i
						// Find the user message before this assistant message
						for j := i - 1; j >= 0; j-- {
							if messages[j].Role == openai.ChatMessageRoleUser && messages[j].Content != "" {
								lastUserIndex = j
								break
							}
						}
						break
					}
				}
			}

			if lastAssistantIndex == -1 {
				return &plugin.ChatCommandResponse{
					Stdout: "Nothing to revert: no assistant answer found",
				}, nil
			}

			// Remove all messages after the last assistant answer
			truncated := messages[:lastAssistantIndex+1]
			params.AgentHandler.SetMessages(truncated)

			// Get the content of the last assistant answer
			lastAssistantMessage := messages[lastAssistantIndex]
			answer := lastAssistantMessage.Content

			// Re-send the last user message to get the same assistant answer
			if lastUserIndex == -1 {
				return &plugin.ChatCommandResponse{
					Stdout: "Nothing to revert: no user message found",
				}, nil
			}

			output := "Reverted to previous answer:\n" + answer

			// Return the responses with the stdout to be displayed
			// Note: we do NOT add the user message back, as /revert is meant to
			// simply show the previous answer without changing the conversation state.
			// If the user wants to continue from this point, they can type a new message.

			return &plugin.ChatCommandResponse{
				Stdout: output,
			}, nil
		},
	})
	return nil
}
