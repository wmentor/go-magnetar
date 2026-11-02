package lastresponse

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

const lastResponseToolName = "last_response"

// ResponseParams represents parameters for the last_response tool.
type ResponseParams struct {
	Index int `json:"index" jsonschema_description:"The index of the response to retrieve (0 = most recent, 1 = second most recent, etc.)"`
}

// Tool provides last_response operation as LLM tool.
type Tool struct {
	cfg           *config.Config
	printer       printer.Printer
	clientFactory requester.ClientFactory
	responseStore plugin.ResponseStore
}

// New creates a new Tool instance.
func New(cfg *config.Config, p printer.Printer, clientFactory requester.ClientFactory, responseStore plugin.ResponseStore) *Tool {
	return &Tool{
		cfg:           cfg,
		printer:       p,
		clientFactory: clientFactory,
		responseStore: responseStore,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return lastResponseToolName
}

// Description returns the tool description.
func (t *Tool) Description() string {
	return `Retrieve a previously stored response from the conversation history.
This tool allows you to access previous answers to the same question or related questions.
Use this tool ONLY when explicitly requested by the user.
For example:
- "What was your last answer?"
- "Show me the previous response about X"
- "What did you say before?"
- "Retrieve the last response"`
}

// Exec executes the last_response tool and returns the result.
func (t *Tool) Exec(ctx context.Context, params *ResponseParams) (any, error) {
	if params == nil {
		return nil, errors.New("invalid params")
	}

	t.printer.ToolCall(printer.IconSearch, lastResponseToolName, "index", params.Index)

	// Validate index
	if params.Index < 0 {
		t.printer.Error(lastResponseToolName+": index must be non-negative", "index", params.Index)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: index must be non-negative",
		}, nil
	}

	// Get response from store
	response, err := t.responseStore.Last(params.Index)
	if err != nil {
		t.printer.Error(lastResponseToolName+": no response found", "index", params.Index, "err", err)
		return &tools.Response{
			Status: tools.StatusOK,
			Result: "error: no response found at the specified index. Try a smaller index value",
		}, nil
	}

	// Format the response with metadata
	result := fmt.Sprintf("Response at index %d (most recent = 0):\n\n%s", params.Index, response)
	return &tools.Response{
		Status: tools.StatusOK,
		Result: result,
	}, nil
}
