package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/invopop/jsonschema"
	"github.com/sashabaranov/go-openai"
)

var ErrToolNotFound = errors.New("tool not found")

type Response struct {
	Status string            `json:"status"`
	Result string            `json:"result,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

// Validatable — интерфейс для ozzo-validation.
type Validatable interface {
	Validate() error
}

type ToolFn func(ctx context.Context, rawJSON []byte) string

type Store struct {
	tools map[string]Tool
	mtx   sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		tools: make(map[string]Tool),
	}
}

func (s *Store) Register[T any](name, description string, execFunc func(ctx context.Context, request *T) (any, error), opts ...ToolOption) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	reflector := &jsonschema.Reflector{
		AllowAdditionalProperties: false, // Обязательно для OpenAI Strict Mode
		DoNotReference:            true,
	}

	schema := reflector.Reflect(new(T))
	schema.AdditionalProperties = nil

	toolDef := func() openai.Tool {
		return openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters:  schema,
				Strict:      true,
			},
		}
	}

	handler := func(ctx context.Context, rawJSON []byte) string {
		var args T

		dec := json.NewDecoder(bytes.NewReader(rawJSON))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&args); err != nil {
			return formatLLMError(translateJSONError(err))
		}

		if validatable, ok := any(&args).(Validatable); ok {
			if err := validatable.Validate(); err != nil {
				if valErr, ok := errors.AsType[validation.Errors](err); ok {
					errMap := make(map[string]string)
					for k, v := range valErr {
						errMap[k] = v.Error()
					}
					return formatLLMError(errMap)
				}
			}
		}

		bizResult, err := execFunc(ctx, &args)
		if err != nil {
			return formatLLMError(map[string]string{"internal_error": err.Error()})
		}

		successJSON, marshalErr := json.Marshal(bizResult)
		if marshalErr != nil {
			return formatLLMError(map[string]string{"internal_error": "failed to marshal response object"})
		}

		return string(successJSON)
	}

	t := Tool{
		Name:       name,
		Definition: toolDef,
		Execute:    handler,
	}

	for _, opt := range opts {
		opt(&t)
	}

	s.tools[name] = t
}

func (s *Store) GetTool(name string) (Tool, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if t, ok := s.tools[name]; ok {
		return t, nil
	}
	return Tool{}, ErrToolNotFound
}

func (s *Store) Tools() []Tool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	result := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		result = append(result, t)
	}
	return result
}

// CloneWith returns a copy of the store containing only the specified tools.
// Returns an error if any of the requested tools are not found.
func (s *Store) CloneWith(toolNames ...string) (*Store, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	cloned := NewStore()

	missing := make([]string, 0)

	for _, name := range toolNames {
		tool, ok := s.tools[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		cloned.tools[name] = tool
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("tools not found: %s", strings.Join(missing, ", "))
	}

	return cloned, nil
}

func translateJSONError(err error) map[string]string {
	errMap := make(map[string]string)
	errStr := err.Error()

	if strings.Contains(errStr, "unknown field") {
		errMap["structure_error"] = "You provided an unsupported parameter. Please check the tool schema and try again."
	} else if strings.Contains(errStr, "cannot unmarshal") {
		errMap["type_error"] = "Data type mismatch (e.g., passed a string instead of an integer)."
	} else {
		errMap["json_error"] = "Invalid JSON syntax."
	}
	return errMap
}

func formatLLMError(errs map[string]string) string {
	res, _ := json.Marshal(map[string]any{ //nolint:errchkjson // ok.
		"status": "validation_failed",
		"errors": errs,
	})
	return string(res)
}
