package gmutex_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/gmutex"
)

func TestMutex(t *testing.T) {
	t.Parallel()

	gmutex.Lock()
	require.False(t, gmutex.TryRLock())
	require.False(t, gmutex.TryLock())
	gmutex.Unlock()

	gmutex.RLock()
	require.False(t, gmutex.TryLock())
	require.True(t, gmutex.TryRLock())
	defer gmutex.RUnlock()
	defer gmutex.RUnlock()
}
