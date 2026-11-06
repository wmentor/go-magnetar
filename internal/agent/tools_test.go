package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAgentName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no hyphens",
			input: "reviewer",
			want:  "reviewer",
		},
		{
			name:  "single hyphen",
			input: "code-reviewer",
			want:  "code_reviewer",
		},
		{
			name:  "multiple hyphens",
			input: "my-complex-agent-name",
			want:  "my_complex_agent_name",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			result := NormalizeAgentName(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}
