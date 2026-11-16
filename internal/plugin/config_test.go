package plugin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	"github.com/wmentor/go-magnetar/internal/plugin/testdata"
)

var _ plugin.Config = (*config.Config)(nil)

func TestStateWithMockConfig(t *testing.T) {
	t.Parallel()

	mockCfg := testdata.NewConfig(t)
	mockCfg.EXPECT().Bool("test.enabled").Return(true)
	mockCfg.EXPECT().String("test.name").Return("magnetar")
	mockCfg.EXPECT().Int("test.count").Return(42)
	mockCfg.EXPECT().Float64("test.ratio").Return(3.14)
	mockCfg.EXPECT().Strings("test.tags").Return([]string{"go", "ai"})
	mockCfg.EXPECT().Profiles().Return([]string{"default", "fast"})
	mockCfg.EXPECT().Profile().Return("default")
	mockCfg.EXPECT().ProfileParamString("model").Return("gpt-4")
	mockCfg.EXPECT().ProfileParamBool("stream").Return(true)
	mockCfg.EXPECT().ProfileParamInt("max_tokens").Return(2048)
	mockCfg.EXPECT().ProfileParamFloat64("temp").Return(0.7)
	mockCfg.EXPECT().SetProfile("fast").Return(nil)
	mockCfg.EXPECT().Copy().Return(&config.Config{})

	state := &plugin.State{
		Config: mockCfg,
	}

	assert.True(t, state.Config.Bool("test.enabled"))
	assert.Equal(t, "magnetar", state.Config.String("test.name"))
	assert.Equal(t, 42, state.Config.Int("test.count"))
	assert.Equal(t, 3.14, state.Config.Float64("test.ratio"))
	assert.Equal(t, []string{"go", "ai"}, state.Config.Strings("test.tags"))
	assert.Equal(t, []string{"default", "fast"}, state.Config.Profiles())
	assert.Equal(t, "default", state.Config.Profile())
	assert.Equal(t, "gpt-4", state.Config.ProfileParamString("model"))
	assert.True(t, state.Config.ProfileParamBool("stream"))
	assert.Equal(t, 2048, state.Config.ProfileParamInt("max_tokens"))
	assert.Equal(t, 0.7, state.Config.ProfileParamFloat64("temp"))
	require.NoError(t, state.Config.SetProfile("fast"))
	assert.NotNil(t, state.Config.Copy())
}
