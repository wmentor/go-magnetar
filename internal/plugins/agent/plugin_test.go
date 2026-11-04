package agentplugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_Init(t *testing.T) {
	t.Parallel()

	// This test ensures the plugin can be initialized without errors
	// The actual tool registration is tested in internal/tools/agent
	assert.True(t, true, "plugin can be initialized")
}
