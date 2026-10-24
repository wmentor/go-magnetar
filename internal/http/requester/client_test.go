package requester_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/http/requester"
)

func TestGet(t *testing.T) {
	t.Parallel()

	for range 10 {
		client := requester.Get()
		require.NotNil(t, client)
	}
}

func TestNewFactory(t *testing.T) {
	t.Parallel()

	factory := requester.NewFactory()
	require.NotNil(t, factory)

	// Verify that factory implements ClientFactory interface
	var _ requester.ClientFactory = factory
}

func TestFactory_Get(t *testing.T) {
	t.Parallel()

	factory := requester.NewFactory()

	// Test that Get() returns a non-nil client
	client := factory.Get()
	require.NotNil(t, client)

	// Test that the returned client can perform HTTP requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFactory_Put(t *testing.T) {
	t.Parallel()

	factory := requester.NewFactory()

	// Test that Put() does not panic
	client := factory.Get()
	require.NotPanics(t, func() {
		factory.Put(client)
	})

	// Test that Put() can be called multiple times
	require.NotPanics(t, func() {
		factory.Put(client)
		factory.Put(nil)
	})
}
